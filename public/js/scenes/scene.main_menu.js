class MainMenu extends Phaser.Scene
{
    connectFailed = false;
    wsConnection = null;

    menuControls = null;
    menuBackground = null;
    loadingSpinner = null;
    connectingText = null;

    game = null;
    onIncomingGameEventCallback = function () {};

    myClientId = null;
    nickname = 'default';
    roomName = 'default';

    constructor ()
    {
        super({ key: 'MainMenu' });
    }

    create ()
    {
        this.loadingSpinner = this.add.sprite(400, 300, 'spinner');
        this.loadingSpinner.setVisible(false);

        this.connectingText = this.make.text({
            x: 10000,
            y: 60,
            text: "Connecting to the server...",
            style: {
                fontFamily: 'Arial',
                color: '#ffffff',
            },
            add: true
        });

        const savedNickname = localStorage.getItem("nickname");
        if (savedNickname) {
            this.nickname = savedNickname;
        } else {
            const defaultNickname = 'Player' + Math.floor(Math.random() * 1000);
            const inputNickname = prompt("Please enter your nickname", defaultNickname);
            if (inputNickname !== null && inputNickname.trim() !== '') {
                this.nickname = inputNickname.trim();
            } else {
                this.nickname = defaultNickname;
            }
            try {
                localStorage.setItem("nickname", this.nickname);
            } catch (e) {
                console.warn("Local storage not available, cannot save nickname");
            }
        }
        // limit nickname up to 10 chars
        this.nickname = this.nickname.substring(0, 10);
        console.log("Using nickname: " + this.nickname);

        this.actionPlay();
    };

    update(game)
    {
        this.loadingSpinner.angle += 2;
    };

    actionPlay()
    {
        console.log('play');
        this.loadingSpinner.setVisible(true);
        this.connectToServer();
    };

    connectToServer()
    {
        if (this.wsConnection) {
            this.wsConnection.send(JSON.stringify({type: 'lobby', subType: 'makeMatch'}));

            return;
        }

        const self = this;
        this.connectingText.x = 0;
        const wsConnect = (nickname) => {
            self.wsConnection = new WebSocket(WEBSOCKET_URL);
            self.wsConnection.onopen = function () {
                self.wsConnection.send(JSON.stringify({type: 'lobby', subType: 'join', data: nickname}));
                self.wsConnection.send(JSON.stringify({type: 'lobby', subType: 'makeMatch', data: {roomName: self.roomName}}));
            };
            self.wsConnection.onclose = () => {
                window.setTimeout(function () {
                    location.reload();
                }, 3000);
            };
            self.wsConnection.onmessage = function (evt) {
                const messages = evt.data.split('\n');
                for (let i = 0; i < messages.length; i++) {
                    let json;
                    try {
                        json = JSON.parse(messages[i]);
                    } catch (ex) {
                        console.warn("Json parse error", evt.data, ex);
                    }
                    if (json) {
                        self.onIncomingMessage(json, evt);
                    }
                }
            };
        };

        wsConnect(this.nickname);
    };

    onIncomingMessage(json, evt)
    {
        if (json.name !== 'CreaturesPosUpdateEvent' && json.name !== 'CreaturesStatsUpdateEvent') {
            console.log('INCOMING', json);
        }

        if (json.name === 'ClientJoinedEvent') {
            this.myClientId = json.data.yourId;
            console.log('My client id = ' + this.myClientId);
            this.connectingText.x = 10000;
            return;
        }
        if (json.name === 'JoinToStartedGameEvent') {
            this.startGame(json.data.gameData);
            return;
        }

        this.onIncomingGameEventCallback(json.name, json.data);
    };

    startGame(gameData)
    {
        const self = this;
        this.scene.start('Game', {
            gameData: gameData,
            sendGameCommand: function (type, data) {
                self.wsConnection.send(JSON.stringify({type: 'game', subType: type, data: data}));
            },
            setOnIncomingGameEventCallback: function (callback) {
                self.onIncomingGameEventCallback = callback;
            },
        });
    };
}

var sceneConfigMainMenu = new MainMenu();
