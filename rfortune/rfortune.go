package rfortune

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var logger = log.New(os.Stdout, "fortune: ", 0)

const (
	PATH_FMT = "fortunes/%s/%s"
	FID_KEY  = "fid"
	MODS_KEY = "fmods"
)

var Pool *redis.Pool

// -----------------------------
type Fortune struct {
	id   int
	mod  string
	text string
}

func (f *Fortune) Path() string {
	return fmt.Sprintf(PATH_FMT, f.id, f.mod)
}

func (f *Fortune) AsHtml() string {
	return fmt.Sprintf("<div id=\"%s\"><pre>%s</pre></div>", f.Path(), f.text)
}
func (f *Fortune) AsText(verbose bool) string {
	if verbose {
		return fmt.Sprintf("%s:%d\n%s", f.mod, f.id, f.text)
	} else {
		return f.text
	}
}

// -----------------------------

// Create a connection pool to the Redis server
func InitRedis(server, password string) {
	Pool = &redis.Pool{
		MaxIdle:     3,
		MaxActive:   20,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
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

// Return a single random Fortune, from a random module
func RandomFortune(mod string) (*Fortune, error) {
	conn := Pool.Get()
	defer conn.Close()

	// ensure the specified module exists
	if mod != "" {
		member, err := redis.Bool(conn.Do("SISMEMBER", MODS_KEY, mod))
		if err != nil {
			return nil, err
		}
		if member == false {
			return nil, errors.New(fmt.Sprintf("module '%s' not found", mod))
		}
	}

	if mod == "" {
		mod2, err := redis.String(conn.Do("SRANDMEMBER", MODS_KEY))
		if err != nil {
			return nil, err
		}
		mod = mod2
	}

	fid, err := redis.Int(conn.Do("SRANDMEMBER", modKey(mod)))
	if err != nil {
		return nil, err
	}

	text, err := redis.String(conn.Do("GET", fortuneKey(fid)))
	if err != nil {
		return nil, err
	}

	return &Fortune{mod: mod, id: fid, text: text}, nil
}

// Load fortune module files, parse, and store in Redis sets
func LoadFortuneMods(dir string) {
	files, _ := filepath.Glob(dir + "/*")

	for _, f := range files {
		var fortunes = loadFortuneMod(f)
		mod := strings.Split(f, "/")[1]
		logger.Printf("Loaded %d fortunes from %s", fortunes.Len(), mod)

		addToRedis(mod, fortunes)
	}
}

func addToRedis(mod string, fortunes list.List) {
	conn := Pool.Get()
	defer conn.Close()

	if fortunes.Len() > 0 {
		_, err := conn.Do("SADD", MODS_KEY, mod)
		checkErr(err, "Error inserting mod")
	}
	for e := fortunes.Front(); e != nil; e = e.Next() {
		// TODO: this screws up the other sends: conn.Send("WATCH", FID_KEY)
		fid, err := redis.Int(conn.Do("INCR", FID_KEY))
		checkErr(err, "Error inserting to redis")
		conn.Send("MULTI")
		conn.Send("SET", fortuneKey(fid), e.Value.(Fortune).text)
		conn.Send("SADD", modKey(e.Value.(Fortune).mod), fid)
		_, err = conn.Do("EXEC")
		checkErr(err, "Error inserting to redis")
	}
}

// Parse the given fortune file and return a list of fortune strings
// Fortune files are delimited by a '%' character on its own line.
func loadFortuneMod(path string) list.List {
	var fortunes list.List

	file, err := os.Open(path)
	checkErr(err, "Can't open file "+path)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	s := ""
	for scanner.Scan() {
		l := scanner.Text()
		if l == "%" {
			fortunes.PushBack(Fortune{mod: strings.Split(path, "/")[1], text: s})
			s = ""
		} else {
			s += l
		}
	}
	return fortunes
}

func ClearFortuneData() {

}

func modKey(mod string) string {
	return "fmod/" + mod
}
func fortuneKey(fid int) string {
	return fmt.Sprintf("f/%d", fid)
}

func checkErr(err error, mesg string) {
	if err != nil {
		logger.Fatalln("%s, %v", mesg, err)
	}
}
