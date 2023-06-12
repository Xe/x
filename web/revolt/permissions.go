package revolt

// Permissions struct
type Permissions struct {
	Bitvise     uint
	Mode        string // can ben CHANNEL, SERVER or USER
	Permissions map[string]uint
}

// Init all of the perms for channel.
func (p *Permissions) InitChannel() *Permissions {
	p.Permissions = map[string]uint{
		"VIEW":            1 << 0,
		"SEND_MESSAGE":    1 << 1,
		"MANAGE_MESSAGES": 1 << 2,
		"MANAGE_CHANNEL":  1 << 3,
		"VOICE_CALL":      1 << 4,
		"INVITE_OTHERS":   1 << 5,
		"EMBED_LINKS":     1 << 6,
		"UPLOAD_FILES":    1 << 7,
	}
	p.Mode = "CHANNEL"
	return p
}

// Init all of the perms for user.
func (p *Permissions) InitUser() *Permissions {
	p.Permissions = map[string]uint{
		"ACCESS":       1 << 0,
		"VIEW_PROFILE": 1 << 1,
		"SEND_MESSAGE": 1 << 2,
		"INVITE":       1 << 3,
	}
	p.Mode = "USER"
	return p
}

// Init all of the perms for server.
func (p *Permissions) InitServer() *Permissions {
	p.Permissions = map[string]uint{
		"VIEW":            1 << 0,
		"MANAGE_ROLES":    1 << 1,
		"MANAGE_CHANNELS": 1 << 2,
		"MANAGE_SERVER":   1 << 3,
		"KICK_MEMBERS":    1 << 4,
		"BAN_MEMBERS":     1 << 5,
		"TIMEOUT_MEMBERS": 1 << 6,
		// 6 bits of space
		"CHANGE_NICKNAME":  1 << 12,
		"CHANGE_NICKNAMES": 1 << 13,
		"CHANGE_AVATAR":    1 << 14,
		"REMOVE_AVATARS":   1 << 15,
	}
	p.Mode = "SERVER"
	return p
}

// Calculate if bitvise has permission
func (p Permissions) Has(perm string) bool {
	if value, ok := p.Permissions[perm]; ok {
		return p.Bitvise&value != 0
	}

	return false
}

// Add new permission(s).
func (p *Permissions) Add(perms ...string) *Permissions {
	for _, perm := range perms {
		if value, ok := p.Permissions[perm]; ok {
			p.Bitvise = p.Bitvise | value
		}
	}

	return p
}

// Remove permission(s).
func (p *Permissions) Remove(perms ...string) *Permissions {
	for _, perm := range perms {
		if value, ok := p.Permissions[perm]; ok {
			p.Bitvise = p.Bitvise - value
		}
	}

	return p
}

// Calculate perms and return unsigned int.
func (p Permissions) Calculate(perms ...string) uint {
	var total uint

	for _, perm := range perms {
		if value, ok := p.Permissions[perm]; ok {
			total = total | value
		}
	}

	return total
}
