package game

import "gorm.io/gorm"

var DB *gorm.DB

type User struct {
	UserId int64 `gorm:"primary_key"`
	Name   string
}
type Game struct {
	Id            int64 `gorm:"primary_key;auto_increment;not_null"`
	Players       uint8
	Turns         []Turn  `gorm:"foreignKey:GameId;references:Id"`
	Winner        []int64 `gorm:"type:integer[]"`
	WinnerGesture uint8
}

type GestureStat struct {
	Gesture uint8
	Count   uint8
}

type Turn struct {
	GameId  int64 `gorm:"primary_key;autoIncrement:false"`
	UserId  int64 `gorm:"primary_key;autoIncrement:false"`
	Gesture uint8
}
