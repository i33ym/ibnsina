package ibnsina

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	m  map[string]string
	mu sync.RWMutex
}

func NewConfig(file *os.File) (*Config, error) {
	config := &Config{
		m: make(map[string]string),
		// sync.Mutex can be used without initialization
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) < 3 {
			continue
		}

		if line[0] == '#' {
			continue
		}

		index := strings.Index(line, "=")
		if index <= 0 {
			continue
		}

		if index == len(line)-1 {
			continue
		}

		config.m[line[:index]] = line[index+1:]
	}

	return config, nil
}

func (config *Config) Log() string {
	config.mu.RLock()
	defer config.mu.RUnlock()

	var buf bytes.Buffer
	for key, value := range config.m {
		if !strings.Contains(key, "PASS") {
			buf.WriteString(key + "=" + value + "\n")
		}
	}

	return buf.String()
}

func (config *Config) String(key string) (string, error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return "", fmt.Errorf("unknown key %s", key)
	}

	return value, nil
}

func (config *Config) StringOrDefault(key string, def string) string {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return def
	}

	return value
}

func (config *Config) MustString(key string) string {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		panic(fmt.Sprintf("unknown key %s !", key))
	}

	return value
}

func (config *Config) SetString(key string, value string) {
	config.mu.Lock()
	defer config.mu.Unlock()

	config.m[key] = value
}

func (config *Config) Int(key string) (int, error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return 0, fmt.Errorf("unknown key %s", key)
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return number, nil
}

func (config *Config) IntOrDefault(key string, def int) int {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return def
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return def
	}

	return number
}

func (config *Config) MustInt(key string) int {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		panic(fmt.Sprintf("unknown key %s !", key))
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("key %q value is not an int", key))
	}

	return number
}

func (config *Config) SetInt(key string, value int) {
	config.mu.Lock()
	defer config.mu.Unlock()

	config.m[key] = strconv.Itoa(value)
}

func (config *Config) Time(key string) (time.Time, error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return time.Time{}, fmt.Errorf("unknown key %s", key)
	}

	time, err := time.Parse(time.UnixDate, value)
	if err != nil {
		return time, err
	}

	return time, nil
}

func (config *Config) TimeOrDefault(key string, def time.Time) time.Time {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return def
	}

	time, err := time.Parse(time.UnixDate, value)
	if err != nil {
		return def
	}

	return time
}

func (config *Config) MustTime(key string) time.Time {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		panic(fmt.Sprintf("unknown key %s", key))
	}

	time, err := time.Parse(time.UnixDate, value)
	if err != nil {
		panic(fmt.Sprintf("key %q value is not a Time", key))
	}

	return time
}

func (config *Config) SetTime(key string, value time.Time) {
	config.mu.Lock()
	defer config.mu.Unlock()

	config.m[key] = value.Format(time.UnixDate)
}

func (config *Config) Bool(key string) (bool, error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return false, fmt.Errorf("unknown key %s", key)
	}

	value = strings.ToLower(value)

	if value == "on" || value == "yes" || value == "enable" {
		value = "true"
	} else if value == "off" || value == "no" || value == "disable" {
		value = "false"
	}

	boolean, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}

	return boolean, nil
}

func (config *Config) BoolOrDefault(key string, def bool) bool {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return def
	}

	value = strings.ToLower(value)

	if value == "on" || value == "yes" || value == "enable" {
		value = "true"
	} else if value == "off" || value == "no" || value == "disable" {
		value = "false"
	}

	boolean, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}

	return boolean
}

func (config *Config) MustBool(key string) bool {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		panic(fmt.Sprintf("unknown key %s", key))
	}

	value = strings.ToLower(value)

	if value == "on" || value == "yes" || value == "enable" {
		value = "true"
	} else if value == "off" || value == "no" || value == "disable" {
		value = "false"
	}

	boolean, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}

	return boolean
}

func (config *Config) SetBool(key string, value bool) {
	str := "false"
	if value {
		str = "true"
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	config.m[key] = str
}

func (config *Config) URL(key string) (*url.URL, error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return nil, fmt.Errorf("unknown key %s", key)
	}

	url, err := url.Parse(value)
	if err != nil {
		return url, err
	}

	return url, nil
}

func (config *Config) URLOrDefault(key string, def *url.URL) *url.URL {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return def
	}

	url, err := url.Parse(value)
	if err != nil {
		return def
	}

	return url
}

func (config *Config) MustURL(key string) *url.URL {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		panic(fmt.Sprintf("unknown key %s", key))
	}

	url, err := url.Parse(value)
	if err != nil {
		panic(fmt.Sprintf("key %q value is not a URL", key))
	}

	return url
}

func (config *Config) SetURL(key string, value *url.URL) {
	config.mu.Lock()
	defer config.mu.Unlock()

	config.m[key] = value.String()
}

func (config *Config) Duration(key string) (time.Duration, error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return time.Duration(0), fmt.Errorf("unknown key %s", key)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return duration, err
	}

	return duration, nil
}

func (config *Config) DurationOrDefault(key string, def time.Duration) time.Duration {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		return def
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return def
	}

	return duration
}

func (config *Config) MustDuration(key string) time.Duration {
	config.mu.RLock()
	defer config.mu.RUnlock()

	value, exists := config.m[key]
	if !exists {
		panic(fmt.Errorf("unknown key %s", key))
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("key %q value is not a Duration", key))
	}

	return duration
}

func (config *Config) SetDuration(key string, value time.Duration) {
	config.mu.Lock()
	defer config.mu.Unlock()

	config.m[key] = value.String()
}
