package main

import (
	"context"
	"fmt"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	kp                     *kingpin.Application
	sec, msgcount, durmute *int
	selfID                 int64
)

const (
	AppID   int = 1957577
	AppHash     = "0ce5138592d80ecbb7ce8fc12ac389b8"
)

type name struct {
	*tg.ChannelFull
}

func init() {
	kp = kingpin.New("AutoMute", "Автоматическое отключение уведомлений в телеграм")
	sec = kp.Flag("sec", "Длительность в секундах за которое будут считаться сообщения (параметр \"count\")").Short('s').Default("15").Int()
	msgcount = kp.Flag("count", "Количество сообщений полученных за \"sec\"").Short('с').Default("3").Int()
	durmute = kp.Flag("durmute", "Время в часах на сколько нужно замьютить").Default("d").Default("1").Int()
}

func main() {
	kp.Parse(os.Args[1:])
	ctx := context.Background()
	if err := run(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(2)
	}
}

func run(ctx context.Context) error {
	dispatcher := tg.NewUpdateDispatcher()
	if client, err := newClient(&dispatcher); err != nil {
		return err
	} else {
		a := termAuth{}
		flow := auth.NewFlow(a, auth.SendCodeOptions{})

		err = client.Run(ctx, func(ctx context.Context) error {
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				return err
			}

			if self, err := client.Self(ctx); err == nil {
				selfID = self.ID
			}

			// Using tg.Client for directly calling RPC.
			raw := tg.NewClient(client)
			if err = Main(raw, &dispatcher); err != nil {
				return err
			}

			fmt.Println("ОК")
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func newClient(dispatcher *tg.UpdateDispatcher) (*telegram.Client, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	sessionDir := filepath.Join(dir, ".td")
	if err := os.MkdirAll(sessionDir, 0600); err != nil {
		return nil, err
	}

	client := telegram.NewClient(AppID, AppHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
		UpdateHandler: dispatcher,
	})

	return client, nil
}

func Main(raw *tg.Client, dispatcher *tg.UpdateDispatcher) error {
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		for _, channel := range e.Channels {
			data, err := raw.ChannelsGetFullChannel(ctx, &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			})
			if err != nil {
				log.Printf("произошла ошибка: %s", err.Error())
				return nil
			}
			chat := data.GetFullChat()
			switch v := chat.(type) {
			case *tg.ChatFull:
				log.Println(channel.Title + " - это чат")
				//if v.NotifySettings.MuteUntil == 0 {
				//	Append(ctx, raw, v).Mute(&tg.InputPeerChat{ChatID: v.ID })
				//}
			case *tg.ChannelFull:
				if v.NotifySettings.MuteUntil == 0 {
					Append(&AutoMute{
						peer: v,
						ctx:  ctx,
						raw:  raw,
						msg:  update.GetMessage().(*tg.Message),
					}).Mute(&tg.InputPeerChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash})
				}
			}
		}

		return nil
	})

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		if msg, ok := update.Message.(*tg.Message); ok && msg.PeerID != nil {
			user, ok := msg.PeerID.(*tg.PeerUser)
			if !ok {
				return nil
			}

			userData, err := raw.UsersGetFullUser(ctx, &tg.InputUser{UserID: user.UserID})
			if err != nil {
				log.Printf("произошла ошибка: %v\n", err)
				return err
			}
			if userData.FullUser.NotifySettings.MuteUntil <= int(time.Now().Unix()) {
				Append(&AutoMute{
					peer: &userData.FullUser,
					ctx:  ctx,
					raw:  raw,
					msg:  msg,
				}).Mute(&tg.InputPeerUser{UserID: userData.FullUser.ID})
			} else if len(userData.Users) == 1 {
				if u, ok := userData.Users[0].(*tg.User); ok {
					fmt.Printf("Пользователь %s %s уже замьючен\n", u.FirstName, u.LastName)
				}
			}
		}

		return nil
	})

	return nil
}

func mute(ctx context.Context, raw *tg.Client, peer tg.InputPeerClass, durmute int) error {
	_, err := raw.AccountUpdateNotifySettings(ctx, &tg.AccountUpdateNotifySettingsRequest{
		Peer: &tg.InputNotifyPeer{Peer: peer},
		Settings: tg.InputPeerNotifySettings{
			ShowPreviews: false,
			Silent:       true,
			MuteUntil:    int(time.Now().Add(time.Hour * time.Duration(durmute)).Unix()),
		},
	})
	if err != nil {
		return err
	}

	return nil
}
