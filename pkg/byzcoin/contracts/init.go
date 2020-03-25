package contracts

import (
	"github.com/calypso-demo/filesharing/pkg/byzcoin"
	"go.dedis.ch/onet/v3/log"
)

func init() {
	err := byzcoin.RegisterGlobalContract(ContractValueID, contractValueFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
	err = byzcoin.RegisterGlobalContract(ContractCoinID, contractCoinFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
	err = byzcoin.RegisterGlobalContract(ContractInsecureDarcID, contractInsecureDarcFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
}
