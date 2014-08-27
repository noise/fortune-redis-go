package rfortune

import (
	"fmt"
	"log"
	"os"
	"container/list"
	"bufio"
	"path/filepath"
	"time"
	"github.com/garyburd/redigo/redis"
)

var logger = log.New(os.Stdout, "fortune: ", 0)

const PATH_FMT = "fortunes/%s/%s"
const FID_KEY = "fid"
const MODS_KEY = "fmods"

// -----------------------------
type Fortune struct {
	id int
	mod string
	text string
}

func (f *Fortune) Path() string {
	return fmt.Sprintf(PATH_FMT, f.id, f.mod)
}

func (f *Fortune) AsHtml() string {
	return fmt.Sprintf("<div id=\"%s\"><pre>%s</pre></div>", f.Path(), f.text)
}
func (f *Fortune) AsText() string {
	return fmt.Sprintf("%s:%d\n%s", f.mod, f.id, f.text)
}
// -----------------------------


var Pool *redis.Pool


func InitRedis(server, password string) {
	Pool = &redis.Pool{
		MaxIdle: 3,
		MaxActive: 20,
		IdleTimeout: 240 * time.Second,
		Dial: func () (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}


func modKey(mod string) string {
	return "fmod/" + mod
}
func fortuneKey(fid int) string {
	return fmt.Sprintf("f/%d", fid)
}

func addToRedis(mod string, fortunes list.List) {
	conn := Pool.Get()
	defer conn.Close()

	if fortunes.Len() > 0 {
		_, err := conn.Do("SADD", MODS_KEY, mod)
		if err != nil {
			logger.Fatal("Error inserting mod", err)
		}
	}
	for e := fortunes.Front(); e != nil; e = e.Next() {
		// TODO: this screws up the other sends: conn.Send("WATCH", FID_KEY)
		fid, err := redis.Int(conn.Do("INCR", FID_KEY))
		if err != nil {
			logger.Fatal("Error inserting to redis", err)
		}			
		conn.Send("MULTI")
		conn.Send("SET", fortuneKey(fid), e.Value.(Fortune).text)
		conn.Send("SADD", modKey(e.Value.(Fortune).mod), fid)
		_, err = conn.Do("EXEC")
		if err != nil {
			logger.Fatal("Error inserting to redis", err)
		}
	}
}

// Parse the given fortune file and return a list of fortune strings
// Fortune files are delimited by a '%' character on its own line.
func loadFortuneMod(path string) list.List {
	var fortunes list.List

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	s := ""
	for scanner.Scan() {
		l := scanner.Text()
		if l == "%" {
			fortunes.PushBack(Fortune{mod: path, text: s})
			s = ""
		} else {
			s += l
		}
	}
	return fortunes
}

func LoadFortuneMods(dir string) {
	files, _ := filepath.Glob(dir + "/*")
	
	for _, f := range files {
		var fortunes = loadFortuneMod(f)
		logger.Printf("Loaded %d fortunes from %s", fortunes.Len(), f)
		addToRedis(f, fortunes)
	}
}	

func RandomFortune(mod string) Fortune {
	conn := Pool.Get()
	defer conn.Close()

	if mod == "" {
		mod2, err := redis.String(conn.Do("SRANDMEMBER", MODS_KEY))
		if (err != nil) {
			logger.Fatal("error fetching mod: ", err)
		} else {
			mod = mod2
		}
	}
	fid, err := redis.Int(conn.Do("SRANDMEMBER", modKey(mod)))
	if (err != nil) {
		logger.Fatal("error fetching fortune id, mod: " + mod, err)
	}

	text, err := redis.String(conn.Do("GET", fortuneKey(fid)))
	if (err != nil) {
		logger.Fatal("error fetching fortune text: ", err)
	}

	return Fortune{mod: mod, id: fid, text: text}
}
