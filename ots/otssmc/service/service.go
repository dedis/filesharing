package service

import (
	"github.com/calypso-demo/filesharing/ots/otsclient/util"
	"github.com/calypso-demo/filesharing/ots/otssmc/protocol"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

const ServiceName = "OTSSMCService"

type OTSSMCService struct {
	*onet.ServiceProcessor
}

type OTSDecryptReq struct {
	RootIndex int
	Roster    *onet.Roster
	Data      *util.OTSDecryptReqData
	Signature *crypto.SchnorrSig
}

type OTSDecryptResp struct {
	DecShares []*util.DecryptedShare
}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

func init() {
	onet.RegisterNewService(ServiceName, newOTSSMCService)
	network.RegisterMessage(&OTSDecryptReq{})
	network.RegisterMessage(&OTSDecryptResp{})
	// network.RegisterMessage(&util.OTSDecryptReqData{})
	// network.RegisterMessage(&util.DecryptedShare{})
}

func (s *OTSSMCService) OTSDecryptReq(req *OTSDecryptReq) (*OTSDecryptResp, onet.ClientError) {
	log.Lvl3("OTSDecryptReq received in service")
	// Tree with depth = 1
	childCount := len(req.Roster.List) - 1
	log.Lvl3("Number of childs:", childCount)
	tree := req.Roster.GenerateNaryTreeWithRoot(childCount, s.ServerIdentity())
	if tree == nil {
		return nil, onet.NewClientErrorCode(ErrorParse, "couldn't create tree")
	}

	pi, err := s.CreateProtocol(protocol.Name, tree)
	if err != nil {
		return nil, onet.NewClientError(err)
	}

	otsDec := pi.(*protocol.OTSDecrypt)
	otsDec.DecReqData = req.Data
	otsDec.Signature = req.Signature
	otsDec.RootIndex = req.RootIndex
	err = pi.Start()
	if err != nil {
		return nil, onet.NewClientError(err)
	}

	resp := &OTSDecryptResp{
		DecShares: <-pi.(*protocol.OTSDecrypt).DecShares,
	}
	return resp, nil
}

func (s *OTSSMCService) NewProtocol(tn *onet.TreeNodeInstance, conf *onet.GenericConfig) (onet.ProtocolInstance, error) {
	log.Lvl3("OTSDecrypt Service received New Protocol event")
	pi, err := protocol.NewProtocol(tn)
	return pi, err
}

func newOTSSMCService(c *onet.Context) onet.Service {
	s := &OTSSMCService{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	err := s.RegisterHandler(s.OTSDecryptReq)
	log.Lvl3("OTSSMC Service registered")
	if err != nil {
		log.ErrFatal(err, "Couldn't register message:")
	}
	return s
}
