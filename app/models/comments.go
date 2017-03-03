package models
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
  "time"
  "github.com/ganggo/federation"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/postgres"
  _ "github.com/jinzhu/gorm/dialects/mssql"
  _ "github.com/jinzhu/gorm/dialects/mysql"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Comment struct {
  ID uint `gorm:"primary_key"`
  CreatedAt time.Time
  UpdatedAt time.Time

  Text string
  ShareableID uint `gorm:"size:4"`
  PersonID uint `gorm:"size:4"`
  Guid string
  LikesCount int `gorm:"size:4"`
  ShareableType string `gorm:"size:60"`
  Signature CommentSignature
}

type Comments []Comment

type CommentSignature struct {
  ID uint `gorm:"primary_key"`
  CreatedAt time.Time
  UpdatedAt time.Time

  CommentId int `gorm:"primary_key;size:4"`
  AuthorSignature string
  // TODO
  //SignatureOrderId int `gorm:"primary_key" gorm:"type:int(4)"`
  AdditionalData string
}

type CommentSignatures []CommentSignature

func (c *Comment) Cast(entity *federation.EntityComment) (err error) {
  db, err := gorm.Open(DB.Driver, DB.Url)
  if err != nil {
    return
  }
  defer db.Close()

  var post Post
  err = db.Where("guid = ?", entity.ParentGuid).First(&post).Error
  if err != nil {
    return
  }
  var person Person
  err = db.Where("diaspora_handle = ?",
    entity.DiasporaHandle).First(&person).Error
  if err != nil {
    return
  }

  (*c).Text = entity.Text
  (*c).ShareableID = post.ID
  (*c).PersonID = person.ID
  (*c).Guid = entity.Guid
  (*c).ShareableType = ShareablePost
  (*c).Signature = CommentSignature{
    AuthorSignature: entity.AuthorSignature,
  }
  return nil
}
