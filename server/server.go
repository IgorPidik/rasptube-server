package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"server/playback"
	"strconv"

	zmq "github.com/go-zeromq/zmq4"
)

type ServerEventType string

const (
	PlaybackTogglePlay ServerEventType = "PLAYBACK_TOGGLE_PLAY"
	PlaybackStop                       = "PLAYBACK_STOP"
	PlaybackPlay                       = "PLAYBACK_PLAY"
	PlaybackNext                       = "PLAYBACK_NEXT"
	PlaybackPrev                       = "PLAYBACK_PREV"
	PlayTrackByID                      = "PLAY_TACK_BY_ID"
	Init                               = "INIT_STATE"
	Ack                                = "ACK"
)

func (s ServerEventType) IsValid() error {
	switch s {
	case PlaybackTogglePlay,
		PlaybackStop,
		PlaybackPlay,
		PlaybackNext,
		PlaybackPrev,
		PlayTrackByID,
		Init,
		Ack:
		return nil
	}
	return errors.New("invalid server event type")
}

type ServerEvent struct {
	Type    ServerEventType
	Payload interface{}
}

type PlayTrackByIDPayload struct {
	TrackID uint32
}

type PlaybackState struct {
	PlaylistID       uint32
	TrackID          uint32
	Playing          bool
	TrackCurrentTime uint32
	TrackTotalTime   uint32
}

type PlaybackData struct {
	State    *PlaybackState
	Playlist *playback.Playlist
}

type Server struct {
	Rep          zmq.Socket
	Pub          zmq.Socket
	PlaybackData *PlaybackData
}

func NewServer(initState *PlaybackData) *Server {
	rep := zmq.NewRep(context.Background())
	pub := zmq.NewPub(context.Background())

	return &Server{
		Rep:          rep,
		Pub:          pub,
		PlaybackData: initState,
	}
}

func (s *Server) Init() <-chan ServerEvent {
	if err := s.Rep.Listen("tcp://*:5559"); err != nil {
		log.Fatalf("could not start rep: %v", err)
	}

	if err := s.Pub.Listen("tcp://*:5563"); err != nil {
		log.Fatalf("could not start pub: %v", err)
	}

	// s.Playback.EnableAutoPlay()
	// s.Playback.AttachMediaPositionChangedCallback(s.publishUpdatedState)
	ch := make(chan ServerEvent)
	go s.handleRequests(ch)
	return ch
}

func (s *Server) handleRequests(ch chan<- ServerEvent) {
	for {
		//  Wait for next request from client
		msg, err := s.Rep.Recv()
		if err != nil {
			log.Fatalf("could not recv request: %v", err)
		}

		log.Printf("received request: [%s]\n", msg.Frames[0])
		eventType := ServerEventType(msg.Frames[0])
		if validErr := eventType.IsValid(); validErr != nil {
			log.Fatal(validErr)
		}

		switch eventType {
		case Init:
			s.sendInitState()
			continue
		case PlayTrackByID:
			if trackID, err := strconv.ParseUint(string(msg.Frames[1]), 10, 32); err == nil {
				ch <- ServerEvent{Type: eventType, Payload: PlayTrackByIDPayload{TrackID: uint32(trackID)}}
			}
		default:
			ch <- ServerEvent{Type: eventType}
		}

		if err := s.Rep.Send(zmq.NewMsgString(Ack)); err != nil {
			log.Fatalf("could not send reply: %v", err)
		}
	}
}

func (s *Server) sendInitState() {
	data, err := json.Marshal(s.PlaybackData)
	if err != nil {
		log.Fatalf("failed to marshal init state: %v", err)
	}

	if msgErr := s.Rep.Send(zmq.NewMsg(data)); msgErr != nil {
		log.Fatalf("failed to send init state: %v", msgErr)
	}
}

func (s *Server) PublishState() {
	data, err := json.Marshal(s.PlaybackData.State)
	if err != nil {
		log.Fatalf("failed to marshal state: %v", err)
	}

	if msgErr := s.Pub.Send(zmq.NewMsg(data)); msgErr != nil {
		log.Fatalf("failed to publish state: %v", msgErr)
	}
}

func (s *Server) Stop() {
	s.Rep.Close()
	s.Pub.Close()
}
