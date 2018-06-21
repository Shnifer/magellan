package main

import (
	. "github.com/Shnifer/magellan/commons"
	. "github.com/Shnifer/magellan/log"
	"github.com/Shnifer/magellan/storage"
	"strconv"
	"strings"
)

func (rd *roomServer) OnCommand(room, role, command string) {
	Log(LVL_DEBUG, "room", room, "role", role, "command", command)

	switch {
	case strings.HasPrefix(command, CMD_ADDBUILDREQ):
		buildStr := strings.TrimPrefix(command, CMD_ADDBUILDREQ)
		b, err := Building{}.Decode([]byte(buildStr))
		if err != nil {
			Log(LVL_ERROR, "Command CMD_ADDBUILDREQ sent strange building: "+buildStr)
			return
		}
		key := b.Type + strconv.Itoa(rd.storage.NextID())

		err = rd.storage.Add(b.GalaxyID, key, buildStr)
		if err != nil {
			Log(LVL_ERROR, "OnCommand: room", room, "role", role, "command", command, ":", err)
			return
		}

		//duplicate in warp
		if b.Type == BUILDING_BEACON {
			b.PlanetID = b.GalaxyID
			b.GalaxyID = WARP_Galaxy_ID
			b.Period = 0
			buildStr = string(b.Encode())
			err = rd.storage.Add(b.GalaxyID, key, buildStr)
			if err != nil {
				Log(LVL_ERROR, "OnCommand: room", room, "role", role, "command", command, ":", err)
				return
			}
		}
	case strings.HasPrefix(command, CMD_DELBUILDREQ):
		fullKey := command[len(CMD_DELBUILDREQ):]
		objKey, err := storage.ReadKey(fullKey)
		if err != nil {
			Log(LVL_ERROR, err)
			return
		}
		err = rd.storage.Remove(objKey)
		if err != nil {
			Log(LVL_ERROR, err)
			return
		}

		//try to delete in warp, if never was -- okey, Keys are unique, we marked as deleted something that will never spawn
		objKey.Area = WARP_Galaxy_ID
		err = rd.storage.Remove(objKey)
		if err != nil {
			Log(LVL_ERROR, err)
			return
		}
	case strings.HasPrefix(command, CMD_LOGGAMEEVENT):
		buf := []byte(command[len(CMD_LOGGAMEEVENT):])
		lge, err := LogGameEvent{}.Decode(buf)
		if err != nil {
			Log(LVL_ERROR, err)
			return
		}
		SaveToStorage(lge.Key, lge.Args, lge.StateFields)
	}
}