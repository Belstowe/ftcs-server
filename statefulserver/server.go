package statefulserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"math/big"

	"github.com/Belstowe/ftcs-server/statefulserver/models"
	"github.com/quic-go/quic-go"
)

type Server struct {
	conn           []quic.Connection
	clientListener quic.Listener
	peerListener   quic.Listener
	masterIndex    int
	sharedState    SharedState
}

func NewServer(addrs ...string) (_ *Server, err error) {
	var tlsConf *tls.Config
	if tlsConf, err = generateTLSConfig(); err != nil {
		return nil, err
	}

	srv := Server{
		conn:        make([]quic.Connection, 0, len(addrs)),
		masterIndex: -1,
		sharedState: NewSharedState(),
	}
	if srv.clientListener, err = quic.ListenAddr("0.0.0.0:5000", tlsConf, nil); err != nil {
		return nil, err
	}
	if srv.peerListener, err = quic.ListenAddr("0.0.0.0:5001", tlsConf, nil); err != nil {
		return nil, err
	}

	tlsConf = &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-ftcs-server"},
	}
	for i := range addrs {
		if conn, err := quic.DialAddr(addrs[i], tlsConf, nil); err == nil {
			srv.conn = append(srv.conn, conn)
		}
	}
	srv.determineMaster()
	return &srv, nil
}

func (s *Server) determineMaster() (err error) {
connLoop:
	for i := range s.conn {
		if s.masterIndex != -1 {
			return nil
		}
		var stream quic.Stream
		if stream, err = s.conn[i].OpenStreamSync(context.Background()); err != nil {
			return err
		}
		enc := gob.NewEncoder(stream)
		if err = enc.Encode(models.IAmMaster{}); err != nil {
			return err
		}
		dec := gob.NewDecoder(stream)
		var res interface{}
		if err = dec.Decode(&res); err != nil {
			return err
		}
		switch res.(type) {
		case models.IAmMaster:
			s.masterIndex = i
			break connLoop
		case models.OK:
			continue connLoop
		default:
			return errors.New("received invalid response from a server")
		}
	}
	return nil
}

func (s *Server) removeConnection(i int) error {
	if i >= len(s.conn) {
		return errors.New("out of range")
	}
	s.conn[i] = s.conn[len(s.conn)-1]
	s.conn = s.conn[:len(s.conn)-1]
	if i == s.masterIndex {
		return s.determineMaster()
	}
	return nil
}

func (s *Server) PeerListen() (err error) {
	var conn quic.Connection
	if conn, err = s.peerListener.Accept(context.Background()); err != nil {
		return err
	}
	var stream quic.Stream
	if stream, err = conn.AcceptStream(context.Background()); err != nil {
		return err
	}
	enc := gob.NewEncoder(stream)
	dec := gob.NewDecoder(stream)
	var res interface{}
	if err = dec.Decode(&res); err != nil {
		return nil
	}
	switch res.(type) {
	case models.AddMe:
		s.conn = append(s.conn, conn)
	case models.IAmMaster:
		switch s.masterIndex {
		case -1:
			if err = enc.Encode(models.IAmMaster{}); err != nil {
				return err
			}
		default:
			if err = enc.Encode(models.OK{}); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

func generateTLSConfig() (_ *tls.Config, err error) {
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(rand.Reader, 1024); err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(2)}
	var certDer []byte
	if certDer, err = x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key); err != nil {
		return nil, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDer})
	var tlsCert tls.Certificate
	if tlsCert, err = tls.X509KeyPair(certPem, keyPem); err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-ftcs-server"},
	}, nil
}
