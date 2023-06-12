package revolt

import "fmt"

// Client cache struct.
type Cache struct {
	Users    []*User    `json:"users"`
	Servers  []*Server  `json:"servers"`
	Channels []*Channel `json:"channels"`
	Members  []*Member  `json:"members"`
}

// Get a channel from cache by Id.
// Will return an empty channel struct if not found.
func (c *Cache) GetChannel(id string) *Channel {
	for _, i := range c.Channels {
		if i.Id == id {
			return i
		}
	}

	return &Channel{}
}

// Get a server from cache by Id.
// Will return an empty server struct if not found.
func (c *Cache) GetServer(id string) *Server {
	for _, i := range c.Servers {
		if i.Id == id {
			return i
		}
	}

	return &Server{}
}

// Get an user from cache by Id.
// Will return an empty user struct if not found.
func (c *Cache) GetUser(id string) *User {
	for _, i := range c.Users {
		if i.Id == id {
			return i
		}
	}

	return &User{}
}

// Get a member from cache by Id.
// Will return an empty member struct if not found.
func (c *Cache) GetMember(id string) *Member {
	for _, i := range c.Members {
		if i.Informations.UserId == id {
			return i
		}
	}

	return &Member{}
}

// Remove a channel from cache by Id.
// Will not delete the channel, just deletes the channel from cache.
// Will change the entire channel cache order!
func (c *Cache) RemoveChannel(id string) error {
	for i, v := range c.Channels {
		if v.Id == id {
			c.Channels[i] = c.Channels[len(c.Channels)-1]
			c.Channels = c.Channels[:len(c.Channels)-1]

			return nil
		}
	}

	return fmt.Errorf("channel not found")
}

// Remove a server from cache by Id.
// Will not delete the server, just deletes the server from cache.
// Will change the entire server cache order!
func (c *Cache) RemoveServer(id string) error {
	for i, v := range c.Servers {
		if v.Id == id {
			c.Servers[i] = c.Servers[len(c.Servers)-1]
			c.Servers = c.Servers[:len(c.Servers)-1]

			return nil
		}
	}

	return fmt.Errorf("server not found")
}

// Remove an user from cache by Id.
// Will not delete the user, just deletes the user from cache.
// Will change the entire user cache order!
func (c *Cache) RemoveUser(id string) error {
	for i, v := range c.Users {
		if v.Id == id {
			c.Users[i] = c.Users[len(c.Users)-1]
			c.Users = c.Users[:len(c.Users)-1]

			return nil
		}
	}

	return fmt.Errorf("user not found")
}

// Remove a member from cache by Id.
// Will not delete the member, just deletes the member from cache.
// Will change the entire member cache order!
func (c *Cache) RemoveMember(id string) error {
	for i, v := range c.Members {
		if v.Informations.UserId == id {
			c.Members[i] = c.Members[len(c.Members)-1]
			c.Members = c.Members[:len(c.Members)-1]

			return nil
		}
	}

	return fmt.Errorf("member not found")
}
