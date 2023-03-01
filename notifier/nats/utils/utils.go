package utils

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/viperx"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
)

// RegisterNATSArgs adds nats flags to the provided FlagSet and binds them to Viper.
func RegisterNATSArgs(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("nats-url", "", "NATS URL")
	viperx.MustBindFlag(v, "nats.url", flags.Lookup("nats-url"))

	flags.String("nats-subject-prefix", "infratographer.events", "NATS subject prefix")
	viperx.MustBindFlag(v, "nats.subject_prefix", flags.Lookup("nats-subject-prefix"))

	flags.String("nats-stream-name", "fertilesoil", "NATS stream name to create if it doesn't already exist")
	viperx.MustBindFlag(v, "nats.stream_name", flags.Lookup("nats-stream-name"))

	flags.String("nats-stream-storage", "file", "NATS new stream storage type (memory or file)")
	viperx.MustBindFlag(v, "nats.stream_storage", flags.Lookup("nats-stream-storage"))

	flags.String("nats-nkey", "", "path to nkey file")
	viperx.MustBindFlag(v, "nats.nkey", flags.Lookup("nats-nkey"))

	flags.String("nats-creds", "", "path to creds file")
	viperx.MustBindFlag(v, "nats.creds", flags.Lookup("nats-creds"))
}

func BuildNATSSubject(v *viper.Viper) string {
	return fmt.Sprintf("%s.%s", v.GetString("nats.subject_prefix"), apiv1.EventSubject)
}

func BuildNATSConnFromArgs(v *viper.Viper) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("fertilesoil"),
	}

	if credsFile := v.GetString("nats.creds"); credsFile != "" {
		opts = append(opts, nats.UserCredentials(credsFile))
	} else if nkeysFile := v.GetString("nats.nkey"); nkeysFile != "" {
		opt, err := nats.NkeyOptionFromSeed(v.GetString("nats.nkey"))
		if err != nil {
			return nil, fmt.Errorf("failed to load nkey: %w", err)
		}

		opts = append(opts, opt)
	} else {
		return nil, errors.New("nats: nats-nkey or nats-creds must be provided")
	}

	return nats.Connect(v.GetString("nats.url"), opts...)
}
