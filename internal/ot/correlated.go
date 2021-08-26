package ot

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/cronokirby/safenum"
	"github.com/taurusgroup/multi-party-sig/internal/params"
	"github.com/taurusgroup/multi-party-sig/pkg/hash"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
)

type CorreOTSendSetup struct {
	_Delta   []byte
	_K_Delta [][]byte
}

type CorreOTSetupSender struct {
	// After setup
	hash              *hash.Hash
	setup             *RandomOTReceiveSetup
	_Delta            []byte
	randomOTReceivers []*RandomOTReceiever
}

func NewCorreOTSetupSender(hash *hash.Hash) *CorreOTSetupSender {
	return &CorreOTSetupSender{hash: hash}
}

type CorreOTSetupSendRound1Message struct {
	msgs []*RandomOTReceiveRound1Message
}

func (r *CorreOTSetupSender) Round1(msg *CorreOTSetupReceiveRound1Message) (*CorreOTSetupSendRound1Message, error) {
	var err error
	r.setup, err = RandomOTSetupReceive(r.hash, &msg.msg)
	if err != nil {
		return nil, err
	}

	r._Delta = make([]byte, params.SecBytes)
	_, _ = rand.Read(r._Delta)

	r.randomOTReceivers = make([]*RandomOTReceiever, params.SecParam)
	var ctr [8]byte
	for i := 0; i < params.SecParam; i++ {
		choice := safenum.Choice((r._Delta[i>>3] >> (i & 0b111)) & 1)
		binary.BigEndian.PutUint64(ctr[:], uint64(i))
		r.randomOTReceivers[i] = NewRandomOTReceiver(r.hash.Fork(&hash.BytesWithDomain{
			TheDomain: "CorreOT Random OT Counter",
			Bytes:     ctr[:],
		}), choice, r.setup)
	}

	msgs := make([]*RandomOTReceiveRound1Message, params.SecParam)
	for i := 0; i < params.SecParam; i++ {
		msgs[i], err = r.randomOTReceivers[i].Round1()
		if err != nil {
			return nil, err
		}
	}

	return &CorreOTSetupSendRound1Message{msgs}, nil
}

type CorreOTSetupSendRound2Message struct {
	msgs []*RandomOTReceiveRound2Message
}

func (r *CorreOTSetupSender) Round2(msg *CorreOTSetupReceiveRound2Message) *CorreOTSetupSendRound2Message {
	msgs := make([]*RandomOTReceiveRound2Message, len(r.randomOTReceivers))
	for i := 0; i < len(msg.msgs) && i < len(r.randomOTReceivers); i++ {
		msgs[i] = r.randomOTReceivers[i].Round2(msg.msgs[i])
	}
	return &CorreOTSetupSendRound2Message{msgs}
}

func (r *CorreOTSetupSender) Round3(msg *CorreOTSetupReceiveRound3Message) (*CorreOTSendSetup, error) {
	K_Delta := make([][]byte, len(r.randomOTReceivers))
	var err error
	for i := 0; i < len(msg.msgs) && i < len(r.randomOTReceivers); i++ {
		K_Delta[i], err = r.randomOTReceivers[i].Round3(msg.msgs[i])
		if err != nil {
			return nil, err
		}
	}
	return &CorreOTSendSetup{_Delta: r._Delta, _K_Delta: K_Delta}, nil
}

type CorreOTReceiveSetup struct {
	_K_0 [][]byte
	_K_1 [][]byte
}

type CorreOTSetupReceiver struct {
	// After setup
	hash            *hash.Hash
	group           curve.Curve
	setup           *RandomOTSendSetup
	randomOTSenders []*RandomOTSender
}

func NewCorreOTSetupReceive(hash *hash.Hash, group curve.Curve) *CorreOTSetupReceiver {
	return &CorreOTSetupReceiver{hash: hash, group: group}
}

type CorreOTSetupReceiveRound1Message struct {
	msg RandomOTSetupSendMessage
}

func (r *CorreOTSetupReceiver) Round1() *CorreOTSetupReceiveRound1Message {
	msg, setup := RandomOTSetupSend(r.hash, r.group)
	r.setup = setup

	r.randomOTSenders = make([]*RandomOTSender, params.SecParam)
	var ctr [8]byte
	for i := 0; i < params.SecParam; i++ {
		binary.BigEndian.PutUint64(ctr[:], uint64(i))
		r.randomOTSenders[i] = NewRandomOTSender(r.hash.Fork(&hash.BytesWithDomain{
			TheDomain: "CorreOT Random OT Counter",
			Bytes:     ctr[:],
		}), setup)
	}

	return &CorreOTSetupReceiveRound1Message{*msg}
}

type CorreOTSetupReceiveRound2Message struct {
	msgs []*RandomOTSendRound1Message
}

func (r *CorreOTSetupReceiver) Round2(msg *CorreOTSetupSendRound1Message) (*CorreOTSetupReceiveRound2Message, error) {
	msgs := make([]*RandomOTSendRound1Message, len(r.randomOTSenders))
	var err error
	for i := 0; i < len(msg.msgs) && i < len(r.randomOTSenders); i++ {
		msgs[i], err = r.randomOTSenders[i].Round1(msg.msgs[i])
		if err != nil {
			return nil, err
		}
	}
	return &CorreOTSetupReceiveRound2Message{msgs}, nil
}

type CorreOTSetupReceiveRound3Message struct {
	msgs []*RandomOTSendRound2Message
}

func (r *CorreOTSetupReceiver) Round3(msg *CorreOTSetupSendRound2Message) (*CorreOTSetupReceiveRound3Message, *CorreOTReceiveSetup, error) {
	msgs := make([]*RandomOTSendRound2Message, len(r.randomOTSenders))
	K_0 := make([][]byte, len(r.randomOTSenders))
	K_1 := make([][]byte, len(r.randomOTSenders))
	for i := 0; i < len(msg.msgs) && i < len(r.randomOTSenders); i++ {
		msgsi, resultsi, err := r.randomOTSenders[i].Round2(msg.msgs[i])
		if err != nil {
			return nil, nil, err
		}
		msgs[i] = msgsi
		K_0[i] = resultsi.Rand0
		K_1[i] = resultsi.Rand1
	}
	return &CorreOTSetupReceiveRound3Message{msgs}, &CorreOTReceiveSetup{_K_0: K_0, _K_1: K_1}, nil
}
