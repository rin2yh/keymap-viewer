// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"

	"github.com/guigui-gui/guigui"

	"github.com/yuuki/keymap-viewer/internal/keymap"
	"github.com/yuuki/keymap-viewer/internal/ui"
	"github.com/yuuki/keymap-viewer/internal/via"
)

func main() {
	probe := flag.Bool("probe", false, "print VIA protocol version and exit")
	listHID := flag.Bool("list-hid", false, "list HID devices matching the Corne VID/PID and exit")
	dump := flag.Bool("dump", false, "dump all keycodes for every layer and exit")
	flag.Parse()

	if *listHID {
		if err := via.ListMatchingDevices(os.Stdout); err != nil {
			log.Fatalf("list-hid: %v", err)
		}
		return
	}

	if *probe {
		client, err := via.Open()
		if err != nil {
			log.Fatalf("open VIA device: %v", err)
		}
		defer client.Close()
		v, err := client.ProtocolVersion()
		if err != nil {
			log.Fatalf("protocol version: %v", err)
		}
		fmt.Printf("VIA protocol version: 0x%04X\n", v)
		layers, err := client.LayerCount()
		if err != nil {
			log.Fatalf("layer count: %v", err)
		}
		fmt.Printf("Layer count: %d\n", layers)
		return
	}

	def, err := keymap.LoadEmbeddedDefinition()
	if err != nil {
		log.Fatalf("load embedded definition: %v", err)
	}

	if *dump {
		client, err := via.Open()
		if err != nil {
			log.Fatalf("open VIA device: %v", err)
		}
		defer client.Close()
		snap, err := via.FetchSnapshot(client, def.Matrix.Rows, def.Matrix.Cols)
		if err != nil {
			log.Fatalf("fetch snapshot: %v", err)
		}
		for layer := range snap.Layers {
			fmt.Printf("=== layer %d ===\n", layer)
			for row := range snap.Rows {
				for col := range snap.Cols {
					fmt.Printf(" %04X", snap.Data[layer][row][col])
				}
				fmt.Println()
			}
		}
		return
	}

	root := ui.NewRoot(def, via.Open)
	op := &guigui.RunOptions{
		Title:         "keymap-viewer",
		WindowSize:    image.Pt(1200, 600),
		WindowMinSize: image.Pt(960, 480),
	}
	if err := guigui.Run(root, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
