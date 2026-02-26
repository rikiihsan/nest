package env

import (
	"bufio"
	"os"
	"strings"
)

func openandset(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		os.Setenv(parts[0], parts[1])
	}
	return nil
}

func Load(path ...string) error {
	if len(path) > 0 {
		for _, item := range path {
			err := openandset(item)
			if err != nil {
				return err
			}
		}
	} else {
		err := openandset(".env")
		if err != nil {
			return err
		}
	}
	return nil
}

func Get(key string, defaults ...string) string {
	v, s := os.LookupEnv(key)
	if s {
		return v
	} else {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return v
	}
}
