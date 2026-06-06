package service_test

import (
	"errors"
	"testing"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/service"
)

type stubRoomRepo struct {
	rooms  []models.Room
	exists map[int]bool
	err    error
}

func (s *stubRoomRepo) List() ([]models.Room, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rooms, nil
}

func (s *stubRoomRepo) Exists(id int) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.exists[id], nil
}

func TestRoomServiceListRooms(t *testing.T) {
	repo := &stubRoomRepo{
		rooms: []models.Room{{ID: 1, Number: "101", Status: "available"}},
	}
	svc := service.NewRoomService(repo)

	rooms, err := svc.ListRooms()
	if err != nil {
		t.Fatalf("ListRooms: %v", err)
	}
	if len(rooms) != 1 || rooms[0].Number != "101" {
		t.Fatalf("unexpected rooms: %+v", rooms)
	}
}

func TestRoomServiceRoomExistsRejectsNonPositiveID(t *testing.T) {
	repo := &stubRoomRepo{exists: map[int]bool{1: true}}
	svc := service.NewRoomService(repo)

	exists, err := svc.RoomExists(0)
	if err != nil {
		t.Fatalf("RoomExists: %v", err)
	}
	if exists {
		t.Fatal("expected false for id 0 without hitting repo")
	}

	exists, err = svc.RoomExists(-1)
	if err != nil || exists {
		t.Fatalf("expected false for negative id, exists=%v err=%v", exists, err)
	}
}

func TestRoomServiceRoomExistsDelegatesToRepo(t *testing.T) {
	repo := &stubRoomRepo{exists: map[int]bool{3: true}}
	svc := service.NewRoomService(repo)

	exists, err := svc.RoomExists(3)
	if err != nil || !exists {
		t.Fatalf("expected room 3 to exist, exists=%v err=%v", exists, err)
	}

	exists, err = svc.RoomExists(99)
	if err != nil || exists {
		t.Fatalf("expected room 99 missing, exists=%v err=%v", exists, err)
	}
}

func TestRoomServiceRoomExistsPropagatesRepoError(t *testing.T) {
	repo := &stubRoomRepo{err: errors.New("db error")}
	svc := service.NewRoomService(repo)

	_, err := svc.RoomExists(1)
	if err == nil {
		t.Fatal("expected repo error")
	}
}
