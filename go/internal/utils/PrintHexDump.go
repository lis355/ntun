package utils

import (
	"fmt"
	"strings"
)

func HexDump(data []byte) string {
	// fmt.Printf("%s", strings.Repeat(" ", 10))
	// for i := range 32 {
	// 	fmt.Printf("%02X ", i)

	// 	if (i+1)%8 == 0 {
	// 		fmt.Print(" ")
	// 	}
	// }
	// fmt.Print("\n")

	// fmt.Printf("%s", strings.Repeat(" ", 10))
	// for i := range 32 {
	// 	fmt.Printf("--")

	// 	if i < 31 {
	// 		fmt.Print("-")
	// 	}

	// 	if (i+1)%8 == 0 &&
	// 		i+1 != 32 {
	// 		fmt.Print("-")
	// 	}
	// }
	// fmt.Print("\n")

	var out strings.Builder

	for i := 0; i < len(data); i += 32 {
		fmt.Fprintf(&out, "%08X| ", i)

		var ascii strings.Builder
		ascii.WriteString("|")

		for j := 0; j < 32; j++ {
			if i+j < len(data) {
				fmt.Fprintf(&out, "%02X ", data[i+j])

				b := data[i+j]
				ascii.WriteByte(ByteToASCIIHexDumpChar(b))
			} else {
				fmt.Fprintf(&out, "   ")
				ascii.WriteByte(' ')
			}

			if (j+1)%8 == 0 &&
				j+1 != 32 {
				fmt.Fprintf(&out, " ")
			}
		}

		out.WriteString(ascii.String())
		fmt.Fprintf(&out, "\n")
	}

	return strings.TrimSpace(out.String())
}

func ByteToASCIIHexDumpChar(b byte) byte {
	if b >= 32 &&
		b <= 126 {
		return b
	} else {
		return '.'
	}
}

func BytesToASCIIHexDumpString(data []byte) string {
	var ascii strings.Builder
	for _, b := range data {
		ascii.WriteByte(ByteToASCIIHexDumpChar(b))
	}

	return ascii.String()
}
