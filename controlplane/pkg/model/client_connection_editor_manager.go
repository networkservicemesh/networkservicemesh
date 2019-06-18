package model

import (
	"fmt"
)

// ClientConnectionEditor is a type to commit changes on ClientConnection
type ClientConnectionEditor struct {
	ClientConnection *ClientConnection

	id              string
	connectionState ClientConnectionState
}

type clientConnectionEditorManager struct {
	domain  clientConnectionDomain
	editors map[string]*ClientConnectionEditor

	allowedTransitions map[ClientConnectionState]map[ClientConnectionState]bool
}

func newClientConnectionEditorManager() clientConnectionEditorManager {
	return clientConnectionEditorManager{
		domain:  newClientConnectionDomain(),
		editors: map[string]*ClientConnectionEditor{},

		allowedTransitions: map[ClientConnectionState]map[ClientConnectionState]bool{
			ClientConnectionReady: {
				ClientConnectionRequesting: true,
				ClientConnectionHealing:    true,
				ClientConnectionClosing:    true,
			},
			ClientConnectionRequesting: {
				ClientConnectionReady:   true,
				ClientConnectionBroken:  true,
				ClientConnectionClosing: true,
			},
			ClientConnectionBroken: {
				ClientConnectionHealing: true,
				ClientConnectionClosing: true,
			},
			ClientConnectionHealing: {
				ClientConnectionRequesting: true,
				ClientConnectionClosing:    true,
			},
			ClientConnectionClosing: {
			},
		},
	}
}

func (m *clientConnectionEditorManager) AddClientConnection(id string, connectionState ClientConnectionState, cc *ClientConnection) (*ClientConnectionEditor, error) {
	cc.id = id
	cc.connectionState = connectionState

	editor := &ClientConnectionEditor{
		ClientConnection: cc.clone().(*ClientConnection),
		id:               id,
		connectionState:  connectionState,
	}

	if err := m.domain.AddClientConnection(cc); err != nil {
		return nil, err
	}

	m.editors[id] = editor

	return m.editors[id], nil
}

func (m *clientConnectionEditorManager) GetClientConnection(id string) *ClientConnection {
	return m.domain.GetClientConnection(id)
}

func (m *clientConnectionEditorManager) GetAllClientConnections() []*ClientConnection {
	return m.domain.GetAllClientConnections()
}

func (m *clientConnectionEditorManager) DeleteClientConnection(id string) error {
	delete(m.editors, id)

	return m.domain.DeleteClientConnection(id)
}

func (m *clientConnectionEditorManager) ChangeClientConnectionState(id string, connectionState ClientConnectionState) (*ClientConnectionEditor, error) {
	cc := m.GetClientConnection(id)
	if cc == nil {
		return nil, fmt.Errorf("trying to change state for not existing connection: %v", id)
	}

	cc.connectionState = connectionState

	if ok := m.domain.CompareAndSwapClientConnection(cc, func(connection *ClientConnection) bool {
		return m.allowedTransitions[connection.connectionState][connectionState]
	}); !ok {
		return nil, fmt.Errorf("trying to perform not allowed state transition: %v", id)
	}

	m.editors[id] = &ClientConnectionEditor{
		ClientConnection: m.editors[id].ClientConnection.clone().(*ClientConnection),
		id:               id,
		connectionState:  connectionState,
	}

	return m.editors[id], nil
}

func (m *clientConnectionEditorManager) ResetClientConnectionChanges(editor *ClientConnectionEditor) {
	editor.ClientConnection = m.domain.GetClientConnection(editor.id)
}

func (m *clientConnectionEditorManager) CommitClientConnectionChanges(editor *ClientConnectionEditor) error {
	if editor != m.editors[editor.id] {
		return fmt.Errorf("using completed editor: %v", editor.id)
	}

	if editor.ClientConnection == nil {
		return fmt.Errorf("trying to commit editor for nil connection: %v", editor.id)
	}

	if editor.connectionState != ClientConnectionRequesting && editor.connectionState != ClientConnectionHealing {
		return fmt.Errorf("trying to commit editor for not Requesting or Healing connection: %v", editor.id)
	}

	editor.ClientConnection.id = editor.id
	editor.ClientConnection.connectionState = editor.connectionState

	m.domain.AddOrUpdateClientConnection(editor.ClientConnection)

	return nil
}

func (m *clientConnectionEditorManager) SetClientConnectionModificationHandler(h *ModificationHandler) func() {
	return m.domain.addHandler(h)
}
