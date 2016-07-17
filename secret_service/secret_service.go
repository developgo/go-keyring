package ss

import (
	"fmt"

	"github.com/godbus/dbus"
)

const (
	serviceName         = "org.freedesktop.secrets"
	servicePath         = "/org/freedesktop/secrets"
	serviceInterface    = "org.freedesktop.Secret.Service"
	collectionInterface = "org.freedesktop.Secret.Collection"
	itemInterface       = "org.freedesktop.Secret.Item"
	sessionInterface    = "org.freedesktop.Secret.Session"
	promptInterface     = "org.freedesktop.Secret.Prompt"

	collectionBasePath = "/org/freedesktop/secrets/collection/"
)

// Secret defines a org.freedesk.Secret.Item secret struct.
type Secret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string `dbus:"content_type"`
}

// NewSecret initializes a new Secret.
func NewSecret(session dbus.ObjectPath, secret string) Secret {
	return Secret{
		Session:     session,
		Parameters:  []byte{},
		Value:       []byte(secret),
		ContentType: "text/plain; charset=utf8",
	}
}

// SecretService is an interface for the Secret Service dbus API.
type SecretService struct {
	*dbus.Conn
	object dbus.BusObject
}

// NewSecretService inializes a new SecretService object.
func NewSecretService() (*SecretService, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	return &SecretService{
		conn,
		conn.Object(serviceName, servicePath),
	}, nil
}

// OpenSession opens a secret service session.
func (s *SecretService) OpenSession() (dbus.BusObject, error) {
	var disregard dbus.Variant
	var sessionPath dbus.ObjectPath
	err := s.object.Call(serviceInterface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&disregard, &sessionPath)
	if err != nil {
		return nil, err
	}

	return s.Object(serviceName, sessionPath), nil
}

// GetCollection returns a collection from a name.
func (s *SecretService) GetCollection(name string) dbus.BusObject {
	return s.Object(serviceName, dbus.ObjectPath(collectionBasePath+name))
}

// Unlock unlocks a collection
func (s *SecretService) Unlock(collection dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	err := s.object.Call(serviceInterface+".Unlock", 0, []dbus.ObjectPath{collection}).Store(&unlocked, &prompt)
	if err != nil {
		return err
	}

	_, v, err := s.handlePrompt(prompt)
	if err != nil {
		return err
	}

	collections := v.Value()
	switch c := collections.(type) {
	case []dbus.ObjectPath:
		unlocked = append(unlocked, c...)
	}

	if len(unlocked) != 1 && unlocked[0] != collection {
		return fmt.Errorf("failed to unlock correct collection")
	}

	return nil
}

// Close closes a secret service dbus session.
func (s *SecretService) Close(session dbus.BusObject) error {
	return session.Call(sessionInterface+".Close", 0).Err
}

// CreateCollection with the supplied label.
func (s *SecretService) CreateCollection(label string) (dbus.BusObject, error) {
	properties := map[string]dbus.Variant{
		collectionInterface + ".Label": dbus.MakeVariant(label),
	}
	var collection, prompt dbus.ObjectPath
	err := s.object.Call(serviceInterface+".CreateCollection", 0, properties, "").
		Store(&collection, &prompt)
	if err != nil {
		return nil, err
	}

	_, v, err := s.handlePrompt(prompt)
	if err != nil {
		return nil, err
	}

	if v.String() != "" {
		collection = dbus.ObjectPath(v.String())
	}

	return s.Object(serviceName, collection), nil
}

// CreateItem creates an item in a collection, with label, attributes and a
// related secret.
func (s *SecretService) CreateItem(collection dbus.BusObject, label string, attributes map[string]string, secret Secret) error {
	properties := map[string]dbus.Variant{
		itemInterface + ".Label":      dbus.MakeVariant(label),
		itemInterface + ".Attributes": dbus.MakeVariant(attributes),
	}

	var item, prompt dbus.ObjectPath
	err := collection.Call(collectionInterface+".CreateItem", 0,
		properties, secret, true).Store(&item, &prompt)
	if err != nil {
		return err
	}

	_, _, err = s.handlePrompt(prompt)
	if err != nil {
		return err
	}

	return nil
}

// handlePrompt checks if a prompt should be handles and handles it by
// triggering the prompt and waiting for the Sercret service daemon to display
// the prompt to the user.
func (s *SecretService) handlePrompt(prompt dbus.ObjectPath) (bool, dbus.Variant, error) {
	if prompt != dbus.ObjectPath("/") {
		err := s.Object(serviceName, prompt).Call(promptInterface+".Prompt", 0, "").Err
		if err != nil {
			return false, dbus.MakeVariant(""), err
		}

		promptSignal := make(chan *dbus.Signal, 1)
		s.Signal(promptSignal)

		signal := <-promptSignal
		switch signal.Name {
		case promptInterface + ".Completed":
			dismissed := signal.Body[0].(bool)
			result := signal.Body[1].(dbus.Variant)
			return dismissed, result, nil
		}

	}

	return false, dbus.MakeVariant(""), nil
}

// SearchItems returns a list of items matching the search object.
func (s *SecretService) SearchItems(collection dbus.BusObject, search interface{}) ([]dbus.ObjectPath, error) {
	var results []dbus.ObjectPath
	err := collection.Call(collectionInterface+".SearchItems", 0, search).Store(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetSecret gets secret from an item in a given session.
func (s *SecretService) GetSecret(itemPath dbus.ObjectPath, session dbus.ObjectPath) (*Secret, error) {
	var secret Secret
	err := s.Object(serviceName, itemPath).Call(itemInterface+".GetSecret", 0, session).Store(&secret)
	if err != nil {
		return nil, err
	}

	return &secret, nil
}

// Delete deletes an item from the collection.
func (s *SecretService) Delete(itemPath dbus.ObjectPath) error {
	var prompt dbus.ObjectPath
	err := s.Object(serviceName, itemPath).Call(itemInterface+".Delete", 0).Store(&prompt)
	if err != nil {
		return err
	}

	_, _, err = s.handlePrompt(prompt)
	if err != nil {
		return err
	}

	return nil
}