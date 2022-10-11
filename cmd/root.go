/*
Copyright © 2022 ItsNotGoodName

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/ItsNotGoodName/x-ipc-viewer/config"
	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
	"github.com/ItsNotGoodName/x-ipc-viewer/mpv"
	"github.com/ItsNotGoodName/x-ipc-viewer/xcursor"
	"github.com/ItsNotGoodName/x-ipc-viewer/xwm"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var cfg config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "x-ipc-viewer",
	Short: "IP camera viewer for X11.",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		x, err := xgb.NewConn()
		if err != nil {
			log.Fatalln(err)
		}
		defer x.Close()

		// Cursor
		cursor, err := xcursor.CreateCursor(x, xcursor.LeftPtr)
		if err != nil {
			log.Fatalln(err)
		}

		// Layout
		var layout mosaic.Layout
		if cfg.Layout.IsAuto() {
			layout = mosaic.NewLayoutGridCount(len(cfg.Windows))
		} else {
			layout = mosaic.NewLayoutManual(cfg.LayoutManualWindows)
		}

		// Manager
		manager, err := xwm.NewManager(x, xproto.Setup(x).DefaultScreen(x), cursor, mosaic.New(layout))
		if err != nil {
			log.Fatalln(err)
		}
		defer manager.Release()
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			manager.Release()
			os.Exit(1)
		}()

		// Windows
		windows := make([]xwm.Window, len(cfg.Windows))
		wg := sync.WaitGroup{}
		for i := range cfg.Windows {
			wg.Add(1)
			go func(i int) {
				// Create X window
				w, err := xwm.CreateXSubWindow(x, manager.WID())
				if err != nil {
					log.Fatal(err)
				}

				// Crate player factory
				pf := mpv.NewPlayerFactory(cfg.Windows[i].Flags, cfg.Player.GPU, cfg.Windows[i].LowLatency)

				// Create player
				p, err := pf(w)
				if err != nil {
					log.Fatal(err)
				}
				p = xwm.NewPlayerCache(p)

				// Create window
				windows[i] = xwm.NewWindow(w, p, cfg.Windows[i].Main, cfg.Windows[i].Sub, cfg.Background)

				wg.Done()
			}(i)
		}
		wg.Wait()

		manager.AddWindows(x, windows)

		xwm.HandleEvent(x, manager)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.x-ipc-viewer.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".x-ipc-viewer" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".x-ipc-viewer")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())

		if err = config.Decode(&cfg); err != nil {
			log.Fatalf("unable to decode into struct: %v", err)
		}
	}
}
