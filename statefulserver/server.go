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
	peerConn            map[uuid.UUID]quic.Connection
	peerHostname        map[uuid.UUID]string
	isPeerConnInitiator map[uuid.UUID]bool
	peerListener        quic.Listener
	clientListener      quic.Listener
	serverID            uuid.UUID
	masterID            uuid.NullUUID
	masterChannel       chan struct{}
	state               models.State
	tlsConfig           *tls.Config
}

func NewServer(addrs ...string) (srv *Server, err error) {
	srv = &Server{
		peerConn:            make(map[uuid.UUID]quic.Connection),
		peerHostname:        make(map[uuid.UUID]string),
		isPeerConnInitiator: make(map[uuid.UUID]bool),
		serverID:            uuid.New(),
		masterID:            uuid.NullUUID{},
		masterChannel:       make(chan struct{}),
		state:               *models.NewState(),
	}
	log.Debug().Msgf("generated id %s", srv.serverID)
	if srv.tlsConfig, err = generateTLSConfig(); err != nil {
		return nil, err
	}
	log.Trace().Msgf("generated tls config %+v", srv.tlsConfig)
	if srv.peerListener, err = quic.ListenAddr("0.0.0.0:5001", srv.tlsConfig, &quic.Config{HandshakeIdleTimeout: time.Second, KeepAlivePeriod: time.Second}); err != nil {
		return nil, err
	}
	go srv.peerListen()
	log.Info().Msg("listening for peers on 0.0.0.0:5001...")
	for _, addr := range addrs {
		log.Debug().Str("address", addr).Msgf("making contact")
		if err = srv.initPeerConnection(addr); err != nil {
			if err.Error() == "Application error 0x0 (remote)" {
				continue
			}
			log.Error().Str("address", addr).Msg(err.Error())
		}
	}
	srv.determineMaster()
	if srv.clientListener, err = quic.ListenAddr("0.0.0.0:5000", srv.tlsConfig, &quic.Config{HandshakeIdleTimeout: time.Second, KeepAlivePeriod: time.Second}); err != nil {
		return nil, err
	}
	log.Info().Msg("listening for clients on 0.0.0.0:5000...")
	return srv, nil
}

func (s *Server) Listen() (err error) {
	var conn quic.Connection
	if conn, err = s.clientListener.Accept(context.Background()); err != nil {
		return err
	}
	log.Debug().Str("type", "client").Msgf("received connection from %v", conn.RemoteAddr())
	go func(conn quic.Connection) {
		var model interface{}
		if model, err = s.recv(conn); err != nil {
			log.Debug().
				Str("type", "client").
				Str("address", conn.LocalAddr().String()).
				Msg(err.Error())
			return
		}
		switch typedModel := model.(type) {
		case models.RequestState:
			s.send(conn, models.SendState{State: s.state})
		default:
			s.send(conn, models.ClientError{Message: fmt.Sprintf("unknown model %T, either random or not yet implemented", typedModel)})
		}
	}(conn)
	return nil
}

func (s *Server) initPeerConnection(addr string) (err error) {
	var conn quic.Connection
	if conn, err = quic.DialAddr(addr, s.tlsConfig, &quic.Config{HandshakeIdleTimeout: time.Second, KeepAlivePeriod: time.Second}); err != nil {
		return err
	}
	log.Debug().Str("address", addr).Msg("successfully dialed")
	if err = s.send(conn, models.Ping{ID: s.serverID}); err != nil {
		return err
	}
	log.Debug().Str("address", addr).Msg("sent ping")
	var model interface{}
	if model, err = s.recv(conn); err != nil {
		return err
	}
	switch typedModel := model.(type) {
	case models.Pong:
		if typedModel.ID == s.serverID {
			log.Debug().Msgf("ignored server %s as id %v was identical (=> it is probably my hostname)", addr, s.serverID)
			conn.CloseWithError(quic.ApplicationErrorCode(0), "")
			break
		}
		if _, ok := s.peerConn[typedModel.ID]; !ok {
			log.Info().Msgf("successfully written %s (id: %v) into cluster!", addr, typedModel.ID)
			s.peerHostname[typedModel.ID] = addr
			s.peerConn[typedModel.ID] = conn
			s.isPeerConnInitiator[typedModel.ID] = true
			go s.listenOnPeer(typedModel.ID)
		}
	default:
		return fmt.Errorf("invalid data received from peer %s: expected models.Pong, got %T", addr, model)
	}
	return nil
}

func (s *Server) listenOnPeer(id uuid.UUID) {
	var model interface{}
	var err error
	for {
		if model, err = s.recv(s.peerConn[id]); err != nil {
			log.Warn().Str("id", id.String()).
				Str("address", s.peerHostname[id]).
				Str("peer-address", s.peerConn[id].RemoteAddr().String()).
				Err(err).
				Msg("one of servers died, waiting for it to recover...")
			delete(s.peerConn, id)
			if s.masterID.UUID == id {
				s.masterID = uuid.NullUUID{}
				s.determineMaster()
			}
			if s.isPeerConnInitiator[id] {
				s.fallbackOnPeer(id)
			}
			return
		}
		switch typedModel := model.(type) {
		case models.Ping:
			s.send(s.peerConn[id], models.Pong{ID: s.serverID})
		case models.Pong:
			break
		case models.AreYouMaster:
			if !s.masterID.Valid || s.serverID == s.masterID.UUID {
				s.send(s.peerConn[id], models.MasterNo{})
				if !s.masterID.Valid {
					s.masterID.Valid = true
					s.masterID.UUID = id
				}
			} else {
				s.send(s.peerConn[id], models.MasterYes{})
			}
		case models.MasterNo:
			s.masterChannel <- struct{}{}
		case models.MasterYes:
			s.masterID.UUID = id
			s.masterID.Valid = true
			s.masterChannel <- struct{}{}
		case models.RequestState:
			s.send(s.peerConn[id], models.StateFromMaster{State: s.state})
		case models.StateFromMaster:
			s.state = typedModel.State
			s.masterID.UUID = id
			s.masterID.Valid = true
		case models.StateToMaster:
			s.state = typedModel.State
			for slaveId, conn := range s.peerConn {
				if slaveId == id {
					continue
				}
				s.send(conn, models.StateFromMaster{State: s.state})
			}
		default:
			break
		}
	}
}

func (s *Server) fallbackOnPeer(id uuid.UUID) {
	var err error
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for range ticker.C {
		if err = s.initPeerConnection(s.peerHostname[id]); err == nil {
			log.Info().Str("id", id.String()).Str("address", s.peerHostname[id]).Msg("woke up!")
			return
		}
		log.Info().Str("id", id.String()).Str("address", s.peerHostname[id]).Msg("still not alive")
	}
}

func (s *Server) determineMaster() {
	if s.masterID.Valid {
		log.Debug().Msgf("master already found, it is %v", s.masterID.UUID)
		s.send(s.peerConn[s.masterID.UUID], models.RequestState{})
		return
	}
	log.Debug().Msg("starting master process")
	s.masterID.UUID = s.serverID
	s.masterID.Valid = true
	for _, conn := range s.peerConn {
		s.send(conn, models.AreYouMaster{})
		<-s.masterChannel
	}
	log.Debug().Msgf("final master is %v", s.masterID.UUID)
	if s.masterID.UUID != s.serverID {
		s.send(s.peerConn[s.masterID.UUID], models.RequestState{})
	}
}

func (s Server) send(conn quic.Connection, model interface{}) (err error) {
	var stream quic.SendStream
	if stream, err = conn.OpenUniStream(); err != nil {
		return err
	}
	// stream.SetWriteDeadline(time.Now().Add(time.Second))
	defer stream.Close()
	encoder := gob.NewEncoder(stream)
	if err = encoder.Encode(&model); err != nil {
		return err
	}
	log.Debug().Str("peer-address", conn.RemoteAddr().String()).Msgf("sent packet %#v", model)
	return nil
}

func (s Server) recv(conn quic.Connection) (model interface{}, err error) {
	var stream quic.ReceiveStream
	if stream, err = conn.AcceptUniStream(context.Background()); err != nil {
		return nil, err
	}
	// stream.SetReadDeadline(time.Now().Add(time.Second))
	decoder := gob.NewDecoder(stream)
	if err = decoder.Decode(&model); err != nil {
		return nil, err
	}
	log.Debug().Str("peer-address", conn.RemoteAddr().String()).Msgf("received packet %#v", model)
	return model, nil
}

func (s *Server) peerListen() {
	var conn quic.Connection
	var err error
	for {
		if conn, err = s.peerListener.Accept(context.Background()); err != nil {
			continue
		}
		log.Debug().Msgf("received connection from %s", conn.RemoteAddr())
		go func(conn quic.Connection) {
			var model interface{}
			var err error
			if model, err = s.recv(conn); err != nil {
				log.Debug().Msg(err.Error())
				return
			}
			switch typedModel := model.(type) {
			case models.Ping:
				if typedModel.ID == s.serverID {
					log.Debug().Msgf("ignored peer %s as id %v was identical (=> it is probably my hostname)", conn.RemoteAddr(), s.serverID)
					conn.CloseWithError(quic.ApplicationErrorCode(0), "")
					return
				}
				if err = s.send(conn, models.Pong{ID: s.serverID}); err != nil {
					return
				}
				if _, ok := s.peerConn[typedModel.ID]; !ok {
					log.Info().Msgf("successfully written %s (id: %v) into cluster!", conn.RemoteAddr(), typedModel.ID)
					s.peerConn[typedModel.ID] = conn
					s.isPeerConnInitiator[typedModel.ID] = false
					s.peerHostname[typedModel.ID] = conn.RemoteAddr().String()
					go s.listenOnPeer(typedModel.ID)
				}
			default:
				log.Debug().Msgf("invalid data received from peer %s: expected models.Pong, got %T", conn.RemoteAddr(), model)
				return
			}
		}(conn)
	}
}

func generateTLSConfig() (_ *tls.Config, err error) {
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(rand.Reader, 1024); err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(2), NotAfter: time.Now().Add(365 * 24 * time.Hour)}
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
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"quic-ftcs-server"},
		InsecureSkipVerify: true,
	}, nil
}
