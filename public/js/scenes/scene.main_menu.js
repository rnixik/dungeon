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

        this.loadingSpinner.setVisible(true);
        this.connectToServer();

        this.displayCharacterCreation();
    };

    update(game)
    {
        this.loadingSpinner.angle += 2;
    };

    displayCharacterCreation()
    {
        this.make.text({
            x: 20,
            y: 100,
            text: "Knight",
            style: {
                fontFamily: 'Arial',
                color: '#aa0000',
            },
            add: true
        }).setInteractive({useHandCursor: true}).on('pointerdown', () => {
            this.wsConnection.send(JSON.stringify({type: 'room', subType: 'setAdditionalProperties', data: {"class": "knight" }}));
        });

        this.make.text({
            x: 100,
            y: 100,
            text: "Rogue",
            style: {
                fontFamily: 'Arial',
                color: '#00aa00',
            },
            add: true
        }).setInteractive({useHandCursor: true}).on('pointerdown', () => {
            this.wsConnection.send(JSON.stringify({type: 'room', subType: 'setAdditionalProperties', data: {"class": "rogue" }}));
        });

        this.make.text({
            x: 180,
            y: 100,
            text: "Wizard",
            style: {
                fontFamily: 'Arial',
                color: '#0000aa',
            },
            add: true
        }).setInteractive({useHandCursor: true}).on('pointerdown', () => {
            this.wsConnection.send(JSON.stringify({type: 'room', subType: 'setAdditionalProperties', data: {"class": "mage" }}));
        });

        this.make.text({
            x: 40,
            y: 300,
            text: "Start",
            style: {
                fontFamily: 'Arial',
                color: '#eeeeee',
            },
            add: true
        }).setInteractive({useHandCursor: true}).on('pointerdown', () => {
            this.wsConnection.send(JSON.stringify({type: 'room', subType: 'startGame'}));
        });
    }

    connectToServer()
    {
        const self = this;
        this.connectingText.x = 0;
        const wsConnect = (nickname) => {
            self.wsConnection = new WebSocket(WEBSOCKET_URL);
            self.wsConnection.onopen = function () {
                console.log('WebSocket connected');
                self.wsConnection.send(JSON.stringify({type: 'lobby', subType: 'join', data: nickname}));
                self.wsConnection.send(JSON.stringify({type: 'lobby', subType: 'makeMatch', data: {roomName: self.roomName}}));
                console.log("Sent commands to join lobby and make/join match");
            };
            self.wsConnection.onclose = () => {
                console.log('WebSocket disconnected');
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
        console.log('Connecting to WebSocket server...');
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
            this.loadingSpinner.setVisible(false);
            this.displayCharacterCreation();
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
        console.log('Starting game scene');
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
