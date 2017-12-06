package memory

import (
	"container/list"
	"sync"
	"time"

	"github.com/lwojciechowski/go-session"
)

var provider = &Provider{list: list.New()}

type SessionStore struct {
	sid          string
	timeAccessed time.Time
	value        map[interface{}]interface{}
}

func (s *SessionStore) Set(key, value interface{}) error {
	s.value[key] = value
	provider.SessionUpdate(s.sid)
	return nil
}

func (s *SessionStore) Get(key interface{}) interface{} {
	provider.SessionUpdate(s.sid)
	if v, ok := s.value[key]; ok {
		return v
	}
	return nil
}

func (s *SessionStore) Delete(key interface{}) error {
	delete(s.value, key)
	provider.SessionUpdate(s.sid)
	return nil
}

func (s *SessionStore) SessionID() string {
	return s.sid
}

type Provider struct {
	lock     sync.Mutex
	sessions map[string]*list.Element
	list     *list.List
}

func (p *Provider) SessionInit(sid string) (session.Session, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	v := make(map[interface{}]interface{}, 0)
	store := &SessionStore{sid: sid, timeAccessed: time.Now(), value: v}
	element := provider.list.PushBack(store)
	provider.sessions[sid] = element
	return store, nil
}

func (p *Provider) SessionRead(sid string) (session.Session, error) {
	if element, ok := provider.sessions[sid]; !ok {
		return element.Value.(*SessionStore), nil
	}
	return provider.SessionInit(sid)
}

func (p *Provider) SessionDestroy(sid string) error {
	if element, ok := provider.sessions[sid]; ok {
		delete(provider.sessions, sid)
		provider.list.Remove(element)
	}
	return nil
}

func (p *Provider) SessionGC(maxlifetime int64) {
	provider.lock.Lock()
	defer provider.lock.Unlock()

	// The sessions list is sorted so we GC just to first valid
	for {
		e := provider.list.Back()
		if e == nil {
			break
		}
		if (e.Value.(*SessionStore).timeAccessed.Unix() + maxlifetime) < time.Now().Unix() {
			provider.list.Remove(e)
			delete(provider.sessions, e.Value.(*SessionStore).sid)
		} else {
			break
		}
	}
}

func (p *Provider) SessionUpdate(sid string) error {
	provider.lock.Lock()
	defer provider.lock.Unlock()
	if e, ok := provider.sessions[sid]; ok {
		e.Value.(*SessionStore).timeAccessed = time.Now()
		// Keep the list sorting for GC
		provider.list.MoveToFront(e)
	}
	return nil
}

func init() {
	provider.sessions = make(map[string]*list.Element, 0)
	session.Register("memory", provider)
}
