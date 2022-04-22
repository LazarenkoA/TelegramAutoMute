package main

import (
	"context"
	"github.com/gotd/td/tg"
	"log"
	"sync"
	"time"
)

type Ipeer interface {
	GetID() int64
}

type AutoMute struct {
	count int
	peer  Ipeer
	ctx   context.Context
	raw   *tg.Client
	msg   *tg.Message
}

var collection sync.Map

func Append(current *AutoMute) *AutoMute {
	startDate := int(time.Now().Add(time.Second * -time.Duration(*sec)).Unix())
	if msg := current.getLastMSGbyUserID(current.peer.GetID(), -1, -1); msg == nil || msg.Date >= startDate {
		return current
	}

	if v, ok := collection.LoadOrStore(current.peer.GetID(), current); ok {
		current = v.(*AutoMute)
		current.count++
	} else {
		current.count = 1
		go current.waitAndDelete(time.Second * time.Duration(*sec))
	}

	return current
}

func (a *AutoMute) Mute(peer tg.InputPeerClass) error {
	if a.count == *msgcount {
		_, err := a.raw.AccountUpdateNotifySettings(a.ctx, &tg.AccountUpdateNotifySettingsRequest{
			Peer: &tg.InputNotifyPeer{Peer: peer},
			Settings: tg.InputPeerNotifySettings{
				ShowPreviews: false,
				Silent:       true,
				MuteUntil:    int(time.Now().Add(time.Hour * time.Duration(*durmute)).Unix()),
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *AutoMute) waitAndDelete(d time.Duration) {
	<-time.After(d)
	collection.Delete(a.peer.GetID())
}

func (a *AutoMute) getLastMSGbyUserID(userID int64, maxDate, minDate int) *tg.Message {
	msg := a.getLastMSG(&tg.InputPeerUser{UserID: userID}, &tg.InputPeerEmpty{}, maxDate, minDate, 1)
	if len(msg) > 0 {
		return msg[0]
	}
	return nil
}

func (a *AutoMute) getLastMSG(Peer tg.InputPeerClass, from tg.InputPeerClass, maxDate, minDate, limit int) (result []*tg.Message) {
	if a.raw == nil {
		return result
	}
	resultSearch, err := a.raw.MessagesSearch(a.ctx, &tg.MessagesSearchRequest{
		Peer:    Peer,
		FromID:  &tg.InputPeerSelf{},
		Filter:  &tg.InputMessagesFilterEmpty{},
		Limit:   limit,
		MaxDate: maxDate,
		MinDate: minDate,
	})
	if err != nil {
		log.Println("Произошла ошибка при получении последнего сообщения")
		return result
	}

	var messages []tg.MessageClass
	switch v := resultSearch.(type) {
	case *tg.MessagesMessagesSlice:
		messages = v.Messages
	case *tg.MessagesChannelMessages:
		messages = v.Messages
	}

	for _, msg := range messages {
		if v, ok := msg.(*tg.Message); ok { //&& v.FromID == nil {
			result = append(result, v)
		}
	}

	return result
}
