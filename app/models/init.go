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
  "errors"
  "github.com/revel/revel"
  "gopkg.in/ganggo/ganggo.v0/app/helpers"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/postgres"
  _ "github.com/jinzhu/gorm/dialects/mssql"
  _ "github.com/jinzhu/gorm/dialects/mysql"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "fmt"
  "regexp"
  "runtime"
)

type BaseController struct {
  *revel.Controller
}

type Database struct {
  Driver string
  Url string
}

const (
  Reshare = "Reshare"
  StatusMessage = "StatusMessage"
  ShareablePost = "Post"
  ShareableComment = "Comment"
)

var DB Database

func OpenDatabase() (*gorm.DB, error) {
  db, err := gorm.Open(DB.Driver, DB.Url)
  if err != nil {
    return db, err
  }
  db.SetLogger(helpers.AppLogWrapper{Name: "gorm"})
  db.LogMode(true)
  return db, err
}

// XXX Actually I wanted to integrate it in a custom revel controller
// like described here https://revel.github.io/manual/controllers.html
// but it will throw me always:
//   panic: NewRoute: Failed to find controller for route path action
func CurrentUser(c *revel.Controller) (User, error) {
  db, err := OpenDatabase()
  if err != nil {
    revel.WARN.Println(err)
    return User{}, err
  }
  defer db.Close()

  accessToken := c.Request.Header.Server.Get("access_token")
  if len(accessToken) > 0 {
    var token OAuthToken
    err := token.FindByToken(accessToken[0])
    if err != nil {
      revel.AppLog.Error("Cannot find token", "error", err)
      return User{}, err
    }
    return token.User, nil
  }

  var session Session
  err = db.Where("token = ?", c.Session["TOKEN"]).First(&session).Error
  if err != nil {
    revel.ERROR.Println(err)
    return User{}, err
  }
  return session.User, nil
}

// BACKEND_ONLY ensures the function
// is not called by the API for example
func BACKEND_ONLY() {
  fpcs := make([]uintptr, 1)
  // get the caller function
  // skip 3 levels to get to the caller
  n := runtime.Callers(3, fpcs)
  if n == 0 {
    return
  }
  caller := runtime.FuncForPC(fpcs[0])
  if caller == nil {
    return
  }
  // get the called function
  n = runtime.Callers(2, fpcs)
  if n == 0 {
    return
  }
  function := runtime.FuncForPC(fpcs[0])
  if function == nil {
    return
  }

  re := regexp.MustCompile(`\/.+\.(.+)\..+$`)
  names := re.FindStringSubmatch(caller.Name())
  if len(names) == 2 && len(names[1]) > 2 {
    if names[1][:3] == "Api" {
      panic(names[0] + " is not allowed calling " + function.Name())
    }
  }
}

func generateTags(model interface{}) (tags Tags, err error) {
  var modelID uint
  var modelType, modelText string
  var modelPublic bool

  if post, ok := model.(*Post); ok {
    modelID = post.ID
    modelType = ShareablePost
    modelText = post.Text
    modelPublic = post.Public
  }
  if comment, ok := model.(*Comment); ok {
    modelID = comment.ID
    modelType = ShareableComment
    modelText = comment.Text
  }

  if modelID == 0 && modelType == "" {
    return tags, errors.New("Unknown model type for generateTags")
  }

  tagNames := helpers.ParseTags(modelText)
  for _, match := range tagNames {
    tags = append(tags, Tag{
      Name: match[1],
      ShareableTaggings: ShareableTaggings{
        ShareableTagging{
          Public: modelPublic,
          ShareableID: modelID,
          ShareableType: modelType,
        },
      },
    })
  }
  return
}

func generateNotifications(model interface{}) (notify Notifications, err error) {
  var personID uint
  var guid, text, dataType string
  if post, ok := model.(*Post); ok {
    guid = post.Guid
    personID = post.PersonID
    text = post.Text
    dataType = ShareablePost
  }
  if comment, ok := model.(*Comment); ok {
    guid = comment.Guid
    personID = comment.PersonID
    text = comment.Text
    dataType = ShareableComment
  }

  if personID == 0 && dataType == "" {
    return notify, errors.New("Unknown data type for generateNotifications")
  }

  mentions := helpers.ParseMentions(text)
  if len(mentions) > 0 {
    revel.Config.SetSection("ganggo")
    localhost, found := revel.Config.String("address")
    if !found {
      return notify, errors.New("No server address configured")
    }

    for _, mention := range mentions {
      if mention[3] == localhost {
        var user User
        err = user.FindByUsername(mention[2])
        if err != nil {
          return notify, err
        }

        notify = append(notify, Notification{
          ShareableType: dataType,
          ShareableGuid: guid,
          UserID: user.ID,
          PersonID: personID,
          Unread: true,
        })
      }
    }
  }
  return
}

func parentIsLocal(postID uint) (user User, found bool) {
  db, err := OpenDatabase()
  if err != nil {
    revel.WARN.Println(err)
    return
  }
  defer db.Close()

  var post Post
  // XXX here we assume every comment is related to post
  // that could be a problem in respect of private messages
  err = db.First(&post, postID).Error
  if err != nil {
    return
  }

  if post.Person.UserID > 0 {
    err = db.First(&user, post.Person.UserID).Error
    if err != nil {
      return
    }
    found = true
    return
  }
  return
}

// This is required since gorm.ModifyColumn only supports postgres engine
// see https://github.com/jinzhu/gorm/blob/0a51f6cdc55d1650d9ed3b4c13026cfa9133b01e/scope.go#L1142
func advancedColumnModify(s *gorm.DB, column, dataType string) {
  var format string
  var scope = s.NewScope(s.Value)

  switch DB.Driver {
    case "postgres":
      format = "ALTER TABLE %v ALTER COLUMN %v TYPE %v"
    case "mysql":
      format = "ALTER TABLE %v MODIFY %v %v"
    case "mssql":
      format = "ALTER TABLE %v ALTER COLUMN %v %v"
    default:
      revel.AppLog.Warn("Database doesn't support alter! Please do it manually",
        "driver", DB.Driver, "table", scope.QuotedTableName(),
        "column", column, "type", dataType)
      return
  }
  // modify column in scope
  scope.Raw(fmt.Sprintf(
    format, scope.QuotedTableName(),
    scope.Quote(column), dataType,
  )).Exec()
}

// Returns different methods of searching
// with regular patterns in a database
func advancedColumnSearch(column, expr string) string {
  switch DB.Driver {
  case "postgres":
    return fmt.Sprintf("%s ~ '%s'", column, expr)
  case "mysql":
    fallthrough
  case "sqlite":
    return fmt.Sprintf("%s regexp '%s'", column, expr)
  default:
    return fmt.Sprintf("%s like '%s'", column, expr)
  }
}

// small helper functions to test
// whether a struct was already loaded
func structLoaded(createAt time.Time) bool {
  var unInitialized time.Time
  return createAt != unInitialized
}
