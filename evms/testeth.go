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
)

// AlethVM is s Evm-interface wrapper around the `testeth` binary, based on Aleth.
type AlethVM struct {
	path string
}

func NewAlethVM(path string) *AlethVM {
	return &AlethVM{
		path: path,
	}
}

// RunStateTest implements the Evm interface
func (evm *AlethVM) RunStateTest(path string, out io.Writer) (string, error) {
	var (
		stderr io.ReadCloser
		err    error
	)
	// ../../testeth -t GeneralStateTests --  --testfile ./statetest1.json --jsontrace {} 2> statetest1_testeth_stderr.jsonl
	cmd := exec.Command(evm.path, "-t", "GeneralStateTests",
		"--", "--testfile", path, "--jsontrace", "{\"disableMemory\":true}")

	if stderr, err = cmd.StdoutPipe(); err != nil {
		return cmd.String(), err
	}
	if err = cmd.Start(); err != nil {
		return cmd.String(), err
	}
	// copy everything to the given writer
	evm.Copy(out, stderr)
	// release resources
	cmd.Wait()
	return cmd.String(), nil
}

func (evm *AlethVM) Name() string {
	return "alethvm"
}

func (vm *AlethVM) Close() {
}

// feed reads from the reader, does some geth-specific filtering and
// outputs items onto the channel
func (evm *AlethVM) Copy(out io.Writer, input io.Reader) {
	var stateRoot stateRoot
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		// Calling bytes means that bytes in 'l' will be overwritten
		// in the next loop. Fine for now though, we immediately marshal it
		data := scanner.Bytes()
		var elem vm.StructLog
		err := json.Unmarshal(data, &elem)
		if err != nil {
			//fmt.Printf("aleth err: %v, line\n\t%v\n", err, string(data))
			continue
		}
		// If the output cannot be marshalled, all fields will be blanks.
		// We can detect that through 'depth', which should never be less than 1
		// for any actual opcode
		if elem.Depth == 0 {
			/*  Most likely one of these:
			{"output":"","gasUsed":"0x2d1cc4","time":233624,"error":"gas uint64 overflow"}
			{"stateRoot": "a2b3391f7a85bf1ad08dc541a1b99da3c591c156351391f26ec88c557ff12134"}
			*/
			if stateRoot.StateRoot == "" {
				json.Unmarshal(data, &stateRoot)
				// Aleth doesn't prefix stateroot
				if r := stateRoot.StateRoot; r != "" {
					stateRoot.StateRoot = fmt.Sprintf("0x%v", r)
				}

			}
			// For now, just ignore these
			continue
		}
		// When geth encounters end of code, it continues anyway, on a 'virtual' STOP.
		// In order to handle that, we need to drop all STOP opcodes.
		if elem.Op == 0x0 {
			continue
		}
		// Parity is missing gasCost, memSize and refund
		elem.GasCost = 0
		elem.MemorySize = 0
		elem.RefundCounter = 0
		jsondata, _ := json.Marshal(elem)
		out.Write(jsondata)
		out.Write([]byte("\n"))
	}
	root, _ := json.Marshal(stateRoot)
	out.Write(root)
	out.Write([]byte("\n"))
}
