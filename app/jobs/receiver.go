package jobs
//
// GangGo Application Server
// Copyright (C) 2017 Lukas Matt <lukas@zauberstuhl.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

import (
  "github.com/revel/revel"
  "github.com/ganggo/ganggo/app/models"
  federation "github.com/ganggo/federation"
  helpers "github.com/ganggo/federation/helpers"
  diaspora "github.com/ganggo/federation/diaspora"
)

type Receiver struct {
  Message diaspora.Message
  Entity diaspora.Entity

  Message2 federation.Message

  Guid string
}

func (receiver Receiver) Run() {
  switch entity := receiver.Message2.Entity().(type) {
  case federation.MessageContact:
    if _, ok := receiver.CheckAuthor(entity.GetAuthor()); ok {
      revel.AppLog.Debug("Starting contact receiver")
      receiver.Contact(entity)
    }
    return // XXX
  }

  // search for sender and check his signature
  person, ok := receiver.CheckAuthor(receiver.Message.Sig.KeyId)
  if !ok || !valid(person, receiver.Message, "") {
    return
  }

  switch entity := receiver.Entity.Data.(type) {
  case diaspora.EntityRetraction:
    if _, ok := receiver.CheckAuthor(entity.Author); ok {
      revel.AppLog.Debug("Starting retraction receiver")
      receiver.Retraction(entity)
    }
  case diaspora.EntityProfile:
    if _, ok := receiver.CheckAuthor(entity.Author); ok {
      revel.AppLog.Debug("Starting profile receiver")
      receiver.Profile(entity)
    }
  case diaspora.EntityReshare:
    if _, ok := receiver.CheckAuthor(entity.Author); ok {
      revel.AppLog.Debug("Starting reshare receiver")
      receiver.Reshare(entity)
    }
  case diaspora.EntityStatusMessage:
    if _, ok := receiver.CheckAuthor(entity.Author); ok {
      revel.AppLog.Debug("Starting status message receiver")
      receiver.StatusMessage(entity)
    }
  case diaspora.EntityComment:
    if person, ok := receiver.CheckAuthor(entity.Author); ok {
      revel.AppLog.Debug("Starting comment receiver")
      // validate author_signature
      if valid(person, entity, receiver.Entity.SignatureOrder) {
        receiver.Comment(entity)
      } else {
        revel.AppLog.Error("invalid sig", "entity", entity)
      }
    }
  case diaspora.EntityLike:
    if person, ok := receiver.CheckAuthor(entity.Author); ok {
      revel.AppLog.Debug("Starting like receiver")
      // validate author_signature
      if valid(person, entity, receiver.Entity.SignatureOrder) {
        receiver.Like(entity)
      }
    }
  default:
    revel.AppLog.Error("No matching entity found", "entity", receiver.Entity)
  }
}

func (receiver *Receiver) CheckAuthor(author string) (models.Person, bool) {
  // Will try fetching author from remote
  // if he doesn't exist locally
  fetch := FetchAuthor{Author: author}; fetch.Run()
  if fetch.Err != nil {
    revel.AppLog.Error("Cannot fetch author", "error", fetch.Err)
  }
  return fetch.Person, fetch.Err == nil
}

func valid(person models.Person, entity federation.Message, order string) bool {
  pubKey, err := helpers.ParseRSAPublicKey(
    []byte(person.SerializedPublicKey))
  if err != nil {
    revel.AppLog.Error(err.Error())
    return false
  }

  // verify sender signature
  var signature federation.Signature
  if !signature.New(entity).Verify(order, pubKey) {
    revel.AppLog.Warn("Signature verification failed", "err", signature.Err)
    return false
  }
  revel.AppLog.Debug("Valid signature", "guid", person.Guid)
  return true
}
