package media_library_test

import (
	"image"
	"image/gif"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conku/gorm"
	"github.com/conku/media_library"
	"github.com/conku/qor/test/utils"
)

var db = utils.TestDB()

type MyFileSystem struct {
	media_library.FileSystem
}

func (MyFileSystem) GetSizes() map[string]*media_library.Size {
	return map[string]*media_library.Size{
		"small1": {20, 10},
		"small2": {20, 10},
		"square": {30, 30},
		"big":    {50, 50},
	}
}

type User struct {
	gorm.Model
	Name   string
	Avatar MyFileSystem
}

func init() {
	if err := db.DropTableIfExists(&User{}).Error; err != nil {
		panic(err)
	}
	db.AutoMigrate(&User{})
	media_library.RegisterCallbacks(db)
}

func TestURLWithoutFile(t *testing.T) {
	user := User{Name: "conku"}

	if got, want := user.Avatar.URL(), ""; got != want {
		t.Errorf(`media_library.Base#URL() == %q, want %q`, got, want)
	}
	if got, want := user.Avatar.URL("big"), ""; got != want {
		t.Errorf(`media_library.Base#URL("big") == %q, want %q`, got, want)
	}
	if got, want := user.Avatar.URL("small1", "small2"), ""; got != want {
		t.Errorf(`media_library.Base#URL("small1", "small2") == %q, want %q`, got, want)
	}
}

func TestURLWithFile(t *testing.T) {
	var filePath string
	user := User{Name: "conku"}

	if avatar, err := os.Open("test/logo.png"); err != nil {
		panic("file doesn't exist")
	} else {
		user.Avatar.Scan(avatar)
	}
	if err := db.Save(&user).Error; err != nil {
		panic(err)
	}

	filePath = user.Avatar.URL()
	if _, err := os.Stat(filepath.Join("public", filePath)); err != nil {
		t.Errorf(`media_library.Base#URL() == %q, it's an invalid path`, filePath)
	}

	styleCases := []struct {
		styles []string
	}{
		{[]string{"big"}},
		{[]string{"small1", "small2"}},
	}
	for _, c := range styleCases {
		filePath = user.Avatar.URL(c.styles...)
		if _, err := os.Stat(filepath.Join("public", filePath)); err != nil {
			t.Errorf(`media_library.Base#URL(%q) == %q, it's an invalid path`, strings.Join(c.styles, ","), filePath)
		}
		if strings.Split(path.Base(filePath), ".")[2] != c.styles[0] {
			t.Errorf(`media_library.Base#URL(%q) == %q, it's a wrong path`, strings.Join(c.styles, ","), filePath)
		}
	}
}

func TestSaveIntoFileSystem(t *testing.T) {
	var user = User{Name: "conku"}
	if avatar, err := os.Open("test/logo.png"); err == nil {
		user.Avatar.Scan(avatar)
		if err := db.Save(&user).Error; err == nil {
			if _, err := os.Stat(filepath.Join("public", user.Avatar.URL())); err != nil {
				t.Errorf("should find saved user avatar")
			}

			var newUser User
			db.First(&newUser, user.ID)
			newUser.Avatar.Scan(`{"CropOptions": {"small1": {"X": 5, "Y": 5, "Height": 10, "Width": 20}, "small2": {"X": 0, "Y": 0, "Height": 10, "Width": 20}}, "Crop": true}`)
			db.Save(&newUser)

			if newUser.Avatar.URL() == user.Avatar.URL() {
				t.Errorf("url should be different after crop")
			}

			file, err := os.Open(filepath.Join("public", newUser.Avatar.URL("small1")))
			if err != nil {
				t.Errorf("Failed open croped image")
			}

			if image, _, err := image.DecodeConfig(file); err == nil {
				if image.Width != 20 || image.Height != 10 {
					t.Errorf("image should be croped successfully")
				}
			} else {
				t.Errorf("Failed to decode croped image")
			}
		} else {
			t.Errorf("should saved user successfully")
		}
	} else {
		panic("file doesn't exist")
	}
}

func TestSaveGifIntoFileSystem(t *testing.T) {
	var user = User{Name: "conku"}
	if avatar, err := os.Open("test/test.gif"); err == nil {
		var frames int
		if g, err := gif.DecodeAll(avatar); err == nil {
			frames = len(g.Image)
		}

		avatar.Seek(0, 0)
		user.Avatar.Scan(avatar)
		if err := db.Save(&user).Error; err == nil {
			if _, err := os.Stat(filepath.Join("public", user.Avatar.URL())); err != nil {
				t.Errorf("should find saved user avatar")
			}

			var newUser User
			db.First(&newUser, user.ID)
			newUser.Avatar.Scan(`{"CropOptions": {"small1": {"X": 5, "Y": 5, "Height": 10, "Width": 20}, "small2": {"X": 0, "Y": 0, "Height": 10, "Width": 20}}, "Crop": true}`)
			db.Save(&newUser)

			if newUser.Avatar.URL() == user.Avatar.URL() {
				t.Errorf("url should be different after crop")
			}

			file, err := os.Open(filepath.Join("public", newUser.Avatar.URL("small1")))
			if err != nil {
				t.Errorf("Failed open croped image")
			}

			if g, err := gif.DecodeAll(file); err == nil {
				if g.Config.Width != 20 || g.Config.Height != 10 {
					t.Errorf("gif should be croped successfully")
				}

				for _, image := range g.Image {
					if image.Rect.Dx() != 20 || image.Rect.Dy() != 10 {
						t.Errorf("gif's frames should be croped successfully, but it is %vx%v", image.Rect.Dx(), image.Rect.Dy())
					}
				}
				if frames != len(g.Image) || frames == 0 {
					t.Errorf("Gif's frames should be same")
				}
			} else {
				t.Errorf("Failed to decode croped gif image")
			}
		} else {
			t.Errorf("should saved user successfully")
		}
	} else {
		panic("file doesn't exist")
	}
}
