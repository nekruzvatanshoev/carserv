package cmd

import (
	"log"
	"os"
	"os/signal"

	"github.com/nekruzvatanshoev/carserv/pkg/carserv/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:   RootCmdName,
	Short: RootCmdShort,
	Long:  RootCmdLong,
}

func Execute() {

	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.AddCommand(ServeCmd)
	viper.BindPFlags(ServeCmd.Flags())
}

var (
	ServeCmd = &cobra.Command{
		Use:   ServeCmdName,
		Short: ServeCmdShort,
		Long:  ServeCmdLong,
		Run:   serveCmdFunc(),
	}
)

func serveCmdFunc() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {

		log.Println("Started serve cmd")

		addr := os.Getenv("SERVER_ADDRESS")
		if len(addr) == 0 {
			addr = ":8080"
		}

		serve := server.NewHTTPServer(addr)

		signalCh := make(chan os.Signal, 1)

		go func() {
			if err := serve.ListenAndServe(); err != nil {
				log.Printf("Shutting down the server...%v", err)
				signalCh <- os.Interrupt

			}
		}()

		signal.Notify(signalCh, os.Interrupt)

		sig := <-signalCh

		log.Printf("Shutdown the server...%s", sig.String())
	}
}
