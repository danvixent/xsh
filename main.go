package main

import (
	_ "time/tzdata"
)

func main() {
	//slog := logrus.New()
	//slog.Out = os.Stdout
	//
	//err := os.Setenv("TZ", "") // Use UTC by default :)
	//if err != nil {
	//	slog.Fatal("failed to set env - ", err)
	//}
	//
	//app := &cli.App{}
	//
	//c := cli.NewCli(app)
	//
	//var licenseKey string
	//
	//var configFile string
	//
	//c.Flags().StringVar(&configFile, "config", "./convoy.json", "Configuration file for convoy")
	//c.Flags().StringVar(&licenseKey, "license-key", "", "Convoy license key")

	//c.AddCommand(version.AddVersionCommand())

	//if err := c.Execute(); err != nil {
	//	slog.Fatal(err)
	//}

}
