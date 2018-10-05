package tun2

import "time"

// Backend is the public state of an individual Connection.
type Backend struct {
	ID     string
	Proto  string
	User   string
	Domain string
	Phi    float32
	Host   string
	Usable bool
}

type backendMatcher func(*Connection) bool

func (s *Server) getBackendsForMatcher(bm backendMatcher) []Backend {
	s.connlock.Lock()
	defer s.connlock.Unlock()

	var result []Backend

	for _, c := range s.conns {
		if !bm(c) {
			continue
		}

		result = append(result, Backend{
			ID:     c.id,
			Proto:  c.conn.LocalAddr().Network(),
			User:   c.user,
			Domain: c.domain,
			Phi:    float32(c.detector.Phi(time.Now())),
			Host:   c.conn.RemoteAddr().String(),
			Usable: c.usable,
		})
	}

	return result
}

// KillBackend forcibly disconnects a given backend but doesn't offer a way to
// "ban" it from reconnecting.
func (s *Server) KillBackend(id string) error {
	s.connlock.Lock()
	defer s.connlock.Unlock()

	for _, c := range s.conns {
		if c.id == id {
			c.cancel()
			return nil
		}
	}

	return ErrNoSuchBackend
}

// GetBackendsForDomain fetches all backends connected to this server associated
// to a single public domain name.
func (s *Server) GetBackendsForDomain(domain string) []Backend {
	return s.getBackendsForMatcher(func(c *Connection) bool {
		return c.domain == domain
	})
}

// GetBackendsForUser fetches all backends connected to this server owned by a
// given user by username.
func (s *Server) GetBackendsForUser(uname string) []Backend {
	return s.getBackendsForMatcher(func(c *Connection) bool {
		return c.user == uname
	})
}

// GetAllBackends fetches every backend connected to this server.
func (s *Server) GetAllBackends() []Backend {
	return s.getBackendsForMatcher(func(*Connection) bool { return true })
}
