package server

import (
	"context"
	"encoding/json"
	"log"
	"server/playback"

	zmq "github.com/go-zeromq/zmq4"
)

const PLAYBACK_TOGGLE_PLAY = "PLAYBACK_TOGGLE_PLAY"
const PLAYBACK_PLAY = "PLAYBACK_PLAY"
const PLAYBACK_STOP = "PLAYBACK_STOP"
const PLAYBACK_NEXT = "PLAYBACK_NEXT"
const PLAYBACK_PREV = "PLAYBACK_PREV"
const INIT_STATE = "INIT_STATE"

const ACK = "ACK"

type PlaybackState struct {
	PlaylistID uint
	TrackID    uint
	Playing    bool
}

type InitState struct {
	State    *PlaybackState
	Playlist *playback.Playlist
}

type Server struct {
	Playback *playback.Playback
	Rep      zmq.Socket
	Pub      zmq.Socket
}

func NewServer(playback *playback.Playback) *Server {
	rep := zmq.NewRep(context.Background())
	pub := zmq.NewPub(context.Background())

	return &Server{
		Rep:      rep,
		Pub:      pub,
		Playback: playback,
	}
}

func (s *Server) Start() {
	if err := s.Rep.Listen("tcp://*:5559"); err != nil {
		log.Fatalf("could not start rep: %v", err)
	}

	if err := s.Pub.Listen("tcp://*:5563"); err != nil {
		log.Fatalf("could not start pub: %v", err)
	}

	for {
		//  Wait for next request from client
		msg, err := s.Rep.Recv()
		if err != nil {
			log.Fatalf("could not recv request: %v", err)
		}

		log.Printf("received request: [%s]\n", msg.Frames[0])

		switch string(msg.Frames[0]) {
		case PLAYBACK_PLAY:
			s.Playback.Play()
		case PLAYBACK_STOP:
			s.Playback.Pause()
		case PLAYBACK_TOGGLE_PLAY:
			s.Playback.TogglePlay()
		case PLAYBACK_NEXT:
			s.Playback.Next()
		case PLAYBACK_PREV:
			s.Playback.Prev()
		case INIT_STATE:
			s.sendInitState()
			continue
		}

		s.publishUpdatedState()

		if err := s.Rep.Send(zmq.NewMsgString(ACK)); err != nil {
			log.Fatalf("could not send reply: %v", err)
		}
	}
}

func (s *Server) getPlaybackState() *PlaybackState {
	trackIndex := s.Playback.State.TrackIndex
	return &PlaybackState{
		PlaylistID: s.Playback.Playlist.ID,
		TrackID:    s.Playback.Playlist.Tracks[trackIndex].ID,
		Playing:    s.Playback.State.Playing,
	}
}

func (s *Server) sendInitState() {
	state := &InitState{
		State:    s.getPlaybackState(),
		Playlist: s.Playback.Playlist,
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Fatalf("failed to marshal init state: %v", err)
	}

	if msgErr := s.Rep.Send(zmq.NewMsg(data)); msgErr != nil {
		log.Fatalf("failed to send init state: %v", msgErr)
	}
}

func (s *Server) publishUpdatedState() {
	state := s.getPlaybackState()
	data, err := json.Marshal(state)
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
