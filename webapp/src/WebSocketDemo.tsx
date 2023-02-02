import React, { useState, useCallback, useEffect, ChangeEvent, FormEvent } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';

export const WebSocketDemo = () => {
    const [answer, setAnswer] = useState('');
    const [serverMessage, setServerMessage] = useState('');

    const { sendMessage, lastMessage, readyState }
        = useWebSocket('ws://localhost:1323/ws', {
            onOpen: () => console.log('opened'),
            shouldReconnect: (closeEvent) => true,
        });

    useEffect(() => {
        if (lastMessage !== null) {
            setServerMessage(lastMessage.data);
        }
    }, [lastMessage]);

    // const handleClickSendMessage = useCallback(() => sendMessage('Hello'), []);
    const handleClickSendMessage = (e: FormEvent) => {
        e.preventDefault();
        if (answer.trim().length === 0) {
            return;
        }
        sendMessage(answer);
    }

    // テキスト入力
    const handleChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
        e.preventDefault();
        setAnswer(e.target.value);
    };

    const connectionStatus = {
        [ReadyState.CONNECTING]: 'Connecting',
        [ReadyState.OPEN]: 'Open',
        [ReadyState.CLOSING]: 'Closing',
        [ReadyState.CLOSED]: 'Closed',
        [ReadyState.UNINSTANTIATED]: 'Uninstantiated',
    }[readyState];

    return (
        <div>
            <h1>Web Socket Demo</h1>
            <form onSubmit={handleClickSendMessage}>
                <textarea
                    value={answer}
                    onChange={handleChange}
                    disabled={readyState !== ReadyState.OPEN}
                ></textarea>
            </form>
            <button
                onClick={handleClickSendMessage}
                disabled={readyState !== ReadyState.OPEN}
            >
                Send to server.
            </button>
            <div>The WebSocket is currently {connectionStatus}</div>
            <div>
                Server message is [{serverMessage}]
            </div>
        </div>
    );
};
