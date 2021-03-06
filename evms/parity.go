// Copyright 2019 Martin Holst Swende
// This file is part of the goevmlab library.
//
// The library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the goevmlab library. If not, see <http://www.gnu.org/licenses/>.

package evms

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/core/vm"
	"io"
	"os/exec"
	"strings"
)

type ParityVM struct {
	path string
}

func NewParityVM(path string) *ParityVM {
	return &ParityVM{
		path: path,
	}
}

func (evm *ParityVM) Name() string {
	return "parity"
}

// RunStateTest implements the Evm interface
func (evm *ParityVM) RunStateTest(path string, out io.Writer) (string, error) {
	var (
		stderr io.ReadCloser
		stdout io.ReadCloser
		err    error
	)
	cmd := exec.Command(evm.path, "--std-json", "state-test", path)
	if stderr, err = cmd.StderrPipe(); err != nil {
		return cmd.String(), err
	}
	// Parity, when there's an error on state root, spits out the error on
	// stdout. So we need to read that aswell
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return cmd.String(), err
	}
	if err = cmd.Start(); err != nil {
		return cmd.String(), err
	}
	// copy everything to the given writer
	evm.Copy(out, stderr)
	// copy everything to the given writer -- this means that the
	// stderr output will come _after_ all stderr data. Which is good.
	evm.Copy(out, stdout)
	// release resources
	cmd.Wait()
	return cmd.String(), nil
}

func (evm *ParityVM) Close() {
}

type parityErrorRoot struct {
	Error string
}

func (evm *ParityVM) Copy(out io.Writer, input io.Reader) {
	var stateRoot stateRoot
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		// Calling bytes means that bytes in 'l' will be overwritten
		// in the next loop. Fine for now though, we immediately marshal it
		data := scanner.Bytes()
		//fmt.Printf("parity data: %v\n", string(data))
		var elem vm.StructLog
		json.Unmarshal(data, &elem)
		// If the output cannot be marshalled, all fields will be blanks.
		// We can detect that through 'depth', which should never be less than 1
		// for any actual opcode
		if elem.Depth == 0 {
			/*  Most likely one of these:
			{"error":"State root mismatch (got: 0xa2b3391f7a85bf1ad08dc541a1b99da3c591c156351391f26ec88c557ff12134, expected: 0x0000000000000000000000000000000000000000000000000000000000000000)","gasUsed":"0x2dc6c0","time":146}
			*/
			if stateRoot.StateRoot == ("") {
				var p parityErrorRoot
				json.Unmarshal(data, &p)

				prefix := `State root mismatch (got: `
				if strings.HasPrefix(p.Error, prefix) {
					root := []byte(strings.TrimPrefix(p.Error, prefix))
					stateRoot.StateRoot = string(root[:66])
				}
			}
			continue
		}
		// When geth encounters end of code, it continues anyway, on a 'virtual' STOP.
		// In order to handle that, we need to drop all STOP opcodes.
		if elem.Op == 0x0 {
			continue
		}
		//fmt.Printf("parity: %v\n", string(data))
		jsondata, _ := json.Marshal(elem)
		out.Write(jsondata)
		out.Write([]byte("\n"))
	}
	if stateRoot.StateRoot != ("") {
		root, _ := json.Marshal(stateRoot)
		out.Write(root)
		out.Write([]byte("\n"))
	}
}

// feed reads from the reader, does some parity-specific filtering and
// outputs items onto the channel
func (evm *ParityVM) feed(input io.Reader, opsCh chan (*vm.StructLog)) {
	defer close(opsCh)
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		// Calling bytes means that bytes in 'l' will be overwritten
		// in the next loop. Fine for now though, we immediately marshal it
		data := scanner.Bytes()
		var elem vm.StructLog
		json.Unmarshal(data, &elem)
		// If the output cannot be marshalled, all fields will be blanks.
		// We can detect that through 'depth', which should never be less than 1
		// for any actual opcode
		if elem.Depth == 0 {
			/*  Most likely one of these:
			{"stateRoot":"0xa2b3391f7a85bf1ad08dc541a1b99da3c591c156351391f26ec88c557ff12134"}
			*/
			fmt.Printf("parity non-op, line is:\n\t%v\n", string(data))
			// For now, just ignore these
			continue
		}
		// When geth encounters end of code, it continues anyway, on a 'virtual' STOP.
		// In order to handle that, we need to drop all STOP opcodes.
		if elem.Op == 0x0 {
			continue
		}
		//fmt.Printf("parity: %v\n", string(data))
		opsCh <- &elem
	}
}
