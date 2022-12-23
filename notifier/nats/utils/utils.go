package utils

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/viperx"

	apiv1 "github.com/infratographer/fertilesoil/api/v1"
)

func RegisterNATSArgs(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("nats-url", "", "NATS URL")
	viperx.MustBindFlag(v, "nats.url", flags.Lookup("nats-url"))

	flags.String("nats-subject-prefix", "infratographer.events", "NATS subject prefix")
	viperx.MustBindFlag(v, "nats.subject_prefix", flags.Lookup("nats-subject-prefix"))

	flags.String("nats-nkey", "", "path to nkey file")
	viperx.MustBindFlag(v, "nats.nkey", flags.Lookup("nats-nkey"))
}

func BuildNATSSubject(v *viper.Viper) string {
	return fmt.Sprintf("%s.%s", v.GetString("nats.subject_prefix"), apiv1.EventSubject)
}

func BuildNATSConnFromArgs(v *viper.Viper) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("fertilesoil"),
	}

	opt, err := nats.NkeyOptionFromSeed(v.GetString("nats.nkey"))
	if err != nil {
		return nil, fmt.Errorf("failed to load nkey: %w", err)
	}

	opts = append(opts, opt)

	return nats.Connect(v.GetString("nats.url"), opts...)
}
