package controllers
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
)

type Post struct {
  *revel.Controller
}

func (p Post) Index(guid string) revel.Result {
  var post models.Post

  user, err := models.CurrentUser(p.Controller)
  if err == nil {
    p.ViewArgs["currentUser"] = user
  }

  err = post.FindByGuidAndUser(guid, user)
  if err != nil {
    return p.NotFound(p.Message("errors.controller.post_not_found"))
  }

  if post.Type == models.Reshare {
    return p.Redirect(Post.Index, post.RootGuid)
  }
  return p.Render(post)
}
