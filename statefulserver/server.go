package statefulserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/Belstowe/ftcs-server/statefulserver/models"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

type Server struct {
	peerConn     map[uuid.UUID]quic.Connection
	peerListener quic.Listener
	serverID     uuid.UUID
	tlsConfig    *tls.Config
}

func NewServer(addrs ...string) (srv *Server, err error) {
	if srv.tlsConfig, err = generateTLSConfig(); err != nil {
		return nil, err
	}
	if srv.peerListener, err = quic.ListenAddr("0.0.0.0:5001", srv.tlsConfig, &quic.Config{HandshakeIdleTimeout: time.Second, KeepAlivePeriod: time.Second}); err != nil {
		return nil, err
	}
	srv.serverID = uuid.New()
	go srv.peerListen()
	for _, addr := range addrs {
		if err = srv.initPeerConnection(addr); err != nil {
			log.Error().Str("address", addr).Msg(err.Error())
		}
	}
	return srv, nil
}

func (s *Server) initPeerConnection(addr string) (err error) {
	var conn quic.Connection
	if conn, err = quic.DialAddr(addr, s.tlsConfig, &quic.Config{HandshakeIdleTimeout: time.Second, KeepAlivePeriod: time.Second}); err != nil {
		return err
	}
	if err = s.send(conn, models.Ping{ID: s.serverID}); err != nil {
		return err
	}
	var model interface{}
	if model, err = s.recv(conn); err != nil {
		return err
	}
	switch typedModel := model.(type) {
	case models.Pong:
		if typedModel.ID == s.serverID {
			log.Debug().Msgf("ignored server %s as id %v was identical (=> it is probably my hostname)", addr, s.serverID)
			break
		}
		if _, ok := s.peerConn[typedModel.ID]; !ok {
			log.Info().Msgf("successfully written %s (id: %v) into cluster!", addr, typedModel.ID)
			s.peerConn[typedModel.ID] = conn
		}
	default:
		return fmt.Errorf("invalid data received from peer %s: expected models.Pong, got %T", addr, model)
	}
	return nil
}

func (s Server) send(conn quic.Connection, model interface{}) (err error) {
	var stream quic.SendStream
	if stream, err = conn.OpenUniStream(); err != nil {
		return err
	}
	defer stream.Close()
	encoder := gob.NewEncoder(stream)
	if err = encoder.Encode(model); err != nil {
		return err
	}
	return nil
}

func (s Server) recv(conn quic.Connection) (model interface{}, err error) {
	var stream quic.ReceiveStream
	if stream, err = conn.AcceptUniStream(context.Background()); err != nil {
		return nil, err
	}
	decoder := gob.NewDecoder(stream)
	if err = decoder.Decode(model); err != nil {
		return nil, err
	}
	return model, nil
}

func (s *Server) peerListen() {
	var conn quic.Connection
	var model interface{}
	var err error
	for {
		if conn, err = s.peerListener.Accept(context.Background()); err != nil {
			continue
		}
		go func() {
			if model, err = s.recv(conn); err != nil {
				return
			}
			switch typedModel := model.(type) {
			case models.Ping:
				if typedModel.ID == s.serverID {
					log.Debug().Msgf("ignored peer %s as id %v was identical (=> it is probably my hostname)", conn.RemoteAddr(), s.serverID)
					return
				}
				if err = s.send(conn, models.Pong{ID: s.serverID}); err != nil {
					return
				}
				if _, ok := s.peerConn[typedModel.ID]; !ok {
					log.Info().Msgf("successfully written %s (id: %v) into cluster!", conn.RemoteAddr(), typedModel.ID)
					s.peerConn[typedModel.ID] = conn
				}
			default:
				return
			}
		}()
	}
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
