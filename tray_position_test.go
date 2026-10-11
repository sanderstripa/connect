package main

import (
 "testing"
 "github.com/wailsapp/wails/v3/pkg/application"
)

func TestTrayMenuPosition(t *testing.T) {
 for _, c := range []struct {anchor application.Point; area application.Rect}{
  {application.Point{X:900,Y:800},application.Rect{Width:1000,Height:900}},
  {application.Point{X:-100,Y:700},application.Rect{X:-1200,Width:1200,Height:850}},
  {application.Point{X:15,Y:15},application.Rect{Width:1000,Height:900}},
 } {
  for _, height := range []int{196,270} {
   p:=trayMenuPosition(c.anchor,c.area,220,height)
   if p.X<c.area.X || p.X+220>c.area.X+c.area.Width || p.Y<c.area.Y || p.Y+height>c.area.Y+c.area.Height {t.Fatal("menu outside monitor",p)}
   if p.Y<c.anchor.Y && p.Y+height>c.anchor.Y-16 {t.Fatal("menu covers icon",p)}
   if p.Y>=c.anchor.Y && p.Y<c.anchor.Y+16 {t.Fatal("menu covers icon below",p)}
  }
 }
}
