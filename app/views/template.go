package views
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
  "regexp"
  "path/filepath"
  "os"
  "github.com/shaoshing/train"
  "github.com/revel/revel"
  "github.com/revel/config"
  "github.com/ganggo/ganggo/app/models"
  "github.com/ganggo/ganggo/app/helpers"
  "html/template"
  "github.com/dchest/captcha"
)

type I18nMessages map[string]map[string]string

var TemplateFuncs = map[string]interface{}{
  // database types
  "IsReshare": func(a string) bool {
    return (a == models.Reshare)
  },
  "IsStatusMessage": func(a string) bool {
    return (a == models.StatusMessage)
  },
  "IsShareablePost": func(a string) bool {
    return (a == models.ShareablePost)
  },
  "IsShareableComment": func(a string) bool {
    return (a == models.ShareableComment)
  },
  "IsShareableLike": func(a string) bool {
    return (a == models.ShareableLike)
  },
  "IsShareableContact": func(a string) bool {
    return (a == models.ShareableContact)
  },
  "LikesByTargetID": func(id uint) []models.Like {
    return likes(id, true)
  },
  "DislikesByTargetID": func(id uint) []models.Like {
    return likes(id, false)
  },
  "PostByGuid": func(guid string) (post models.Post) {
    db, err := models.OpenDatabase()
    if err != nil {
      revel.ERROR.Println(err)
      return
    }
    defer db.Close()

    err = db.Where("guid = ?", guid).First(&post).Error
    if err != nil {
      revel.ERROR.Println(err, guid)
      return
    }
    return
  },
  "PersonByID": func(id uint) (person models.Person) {
    db, err := models.OpenDatabase()
    if err != nil {
      revel.ERROR.Println(err)
      return
    }
    defer db.Close()

    err = db.First(&person, id).Error
    if err != nil {
      revel.ERROR.Println(err, id)
      return
    }

    err = db.Where("person_id = ?", person.ID).First(&person.Profile).Error
    if err != nil {
      revel.ERROR.Println(err, person)
      return
    }
    return
  },
  // string parse helper
  "HostFromHandle": func(handle string) (host string) {
    _, host, err := helpers.ParseAuthor(handle)
    if err != nil {
      revel.ERROR.Println(err)
      return
    }
    return
  },
  "FetchUserStreams": func(user models.User) (streams models.UserStreams) {
    db, err := models.OpenDatabase()
    if err != nil {
      revel.AppLog.Error(err.Error())
      return
    }
    err = db.Where("user_id = ?", user.ID).Find(&streams).Error
    if err != nil {
      revel.AppLog.Error(err.Error())
    }
    return
  },
  // captcha generator
  "CaptchaNew": func() string { return captcha.New() },
  "ParseLocalesToJson": func() (i18n I18nMessages) {
    i18n = make(I18nMessages)
    directory := filepath.Join(revel.BasePath, "messages")
    re := regexp.MustCompile(`ganggo\.([\w-_]{1,})$`)
    if err := filepath.Walk(directory, func(path string, f os.FileInfo, err error) error {
      result := re.FindAllStringSubmatch(path, 1)
      if len(result) > 0 && len(result[0]) > 0 {
        c, err := config.ReadDefault(path)
        if err != nil {
          revel.AppLog.Error("Cannot open config file", "err", err)
          return err
        }
        options, err := c.Options(config.DefaultSection)
        if err != nil {
          revel.AppLog.Error("Cannot open config file", "err", err)
          return err
        }
        i18n[result[0][1]] = make(map[string]string)
        for _, option := range options {
          value, _ := c.String(config.DefaultSection, option)
          i18n[result[0][1]][option] = value
        }
      }
      return nil
    }); err != nil {
      revel.AppLog.Error(err.Error())
      return nil
    }
    return i18n
  },
  "FindAvailableLocales": func() (list []string) {
    directory := filepath.Join(revel.BasePath, "messages")
    re := regexp.MustCompile(`ganggo\.([\w-_]{1,})$`)
    err := filepath.Walk(directory, func(path string, f os.FileInfo, err error) error {
      result := re.FindAllStringSubmatch(path, 1)
      if len(result) > 0 && len(result[0]) > 0 {
        list = append(list, result[0][1])
      }
      return nil
    })
    if err != nil {
      revel.ERROR.Println(err)
      return
    }
    return
  },
  "FindUnreadNotifications": func(user models.User) (notify models.Notifications) {
    err := notify.FindUnreadByUserID(user.ID)
    if err != nil {
      revel.ERROR.Println(err)
      return
    }
    return
  },
  "CommentGuid": func(comment models.Comment) string {
    post, _ := comment.ParentPost()
    return post.Guid + "#" + comment.Guid
  },
  "LikeGuid": func(like models.Like) string {
    post, _ := like.ParentPost()
    return post.Guid
  },
  "PtrToValue": func(a *string) string { return *a },
  // custom train script/stylesheet include functions
  "javascript_include_tag": func(name string) template.HTML {
    path := "/assets/javascripts/" + name + ".js"
    src := loadAndFetchManifestEntry(path)
    if src == "" {
      src = "/assets/vendor/js/" + name + ".js"
    }
    tmpl := `<script src="/public` + src + `"></script>`
    return template.HTML(tmpl)
  },
  "stylesheet_link_tag": func(name string) template.HTML {
    path := "/assets/stylesheets/" + name + ".css"
    src := loadAndFetchManifestEntry(path)
    if src == "" {
      src = "/assets/vendor/css/" + name + ".css"
    }
    tmpl := `<link type="text/css" rel="stylesheet" href="/public` + src + `">`
    return template.HTML(tmpl)
  },
  "ugt": func(a, b uint) bool {
    return a > b
  },
  "eq": func(a, b interface {}) bool {
    return a == b
  },
  "ne": func(a, b interface {}) bool {
    return a != b
  },
  "add": func(a, b int) int {
    return a + b
  },
  "uadd": func(a, b uint) uint {
    return a + b
  },
  "sub": func(a, b int) int {
    return a - b
  },
  "usub": func(a, b uint) uint {
    return a - b
  },
  "concat": func(a, b string) string {
    return a + b
  },
  "htmlToString": func(a template.HTML) string {
    return string(a)
  },
  "nilValue": func(a interface {}) bool {
    return a == nil
  },
}

func loadAndFetchManifestEntry(path string) (src string) {
  if len(train.ManifestInfo) <= 0 {
    train.Config.PublicPath = "src/" + revel.ImportPath + "/public"
    err := train.LoadManifestInfo()
    if err != nil {
      revel.ERROR.Println(err)
      return
    }
  }
  src = train.ManifestInfo[path]
  return
}

func likes(id uint, like bool) (likes []models.Like) {
  db, err := models.OpenDatabase()
  if err != nil {
    revel.ERROR.Println(err)
    return
  }
  defer db.Close()

  err = db.Where(
    `target_type = ?
      and target_id = ?
      and positive = ?`,
    models.ShareablePost, id, like,
  ).Find(&likes).Error
  if err != nil {
    revel.ERROR.Println(err)
    return
  }
  return
}
