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
    };

    update(game)
    {
        this.loadingSpinner.angle += 2;
    };

    displayCharacterCreation()
    {
        const centerX = this.cameras.main.width / 2;
        const startY = 150;
        const spacing = 150;

        // Title
        this.make.text({
            x: centerX,
            y: 50,
            text: "Choose Your Hero",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '32px',
                color: '#ffffff',
                stroke: '#000000',
                strokeThickness: 4
            },
            add: true
        });

        // Knight
        const knightSprite = this.add.sprite(centerX - spacing, startY, 'knight');
        knightSprite.setScale(2.5);
        knightSprite.play('knight_idle');
        knightSprite.setInteractive({useHandCursor: true});
        
        const knightText = this.make.text({
            x: centerX - spacing,
            y: startY + 80,
            text: "Knight",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '24px',
                color: '#ff6b6b',
                stroke: '#000000',
                strokeThickness: 3
            },
            add: true
        });
        
        knightSprite.on('pointerdown', () => {
            this.selectCharacter('knight');
            this.highlightSelected(knightSprite, knightText);
        });
        knightSprite.on('pointerover', () => {
            knightSprite.setScale(2.7);
        });
        knightSprite.on('pointerout', () => {
            knightSprite.setScale(2.5);
        });

        // Rogue (Archer)
        const rogueSprite = this.add.sprite(centerX, startY, 'rogue');
        rogueSprite.setScale(2.5);
        rogueSprite.play('rogue_idle');
        rogueSprite.setInteractive({useHandCursor: true});
        
        const rogueText = this.make.text({
            x: centerX,
            y: startY + 80,
            text: "Rogue",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '24px',
                color: '#51cf66',
                stroke: '#000000',
                strokeThickness: 3
            },
            add: true
        });
        
        rogueSprite.on('pointerdown', () => {
            this.selectCharacter('rogue');
            this.highlightSelected(rogueSprite, rogueText);
        });
        rogueSprite.on('pointerover', () => {
            rogueSprite.setScale(2.7);
        });
        rogueSprite.on('pointerout', () => {
            rogueSprite.setScale(2.5);
        });

        // Mage
        const mageSprite = this.add.sprite(centerX + spacing, startY, 'mage');
        mageSprite.setScale(2.5);
        mageSprite.play('mage_idle');
        mageSprite.setInteractive({useHandCursor: true});
        
        const mageText = this.make.text({
            x: centerX + spacing,
            y: startY + 80,
            text: "Wizard",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '24px',
                color: '#74c0fc',
                stroke: '#000000',
                strokeThickness: 3
            },
            add: true
        });
        
        mageSprite.on('pointerdown', () => {
            this.selectCharacter('mage');
            this.highlightSelected(mageSprite, mageText);
        });
        mageSprite.on('pointerover', () => {
            mageSprite.setScale(2.7);
        });
        mageSprite.on('pointerout', () => {
            mageSprite.setScale(2.5);
        });

        // Store references for highlighting
        this.characterSprites = [
            { sprite: knightSprite, text: knightText },
            { sprite: rogueSprite, text: rogueText },
            { sprite: mageSprite, text: mageText }
        ];

        // Start button
        const startButton = this.make.text({
            x: centerX,
            y: startY + 180,
            text: "START GAME",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '28px',
                color: '#ffec99',
                stroke: '#000000',
                strokeThickness: 4
            },
            add: true
        }).setInteractive({useHandCursor: true});
        
        startButton.on('pointerdown', () => {
            this.wsConnection.send(JSON.stringify({type: 'room', subType: 'startGame'}));
        });
        startButton.on('pointerover', () => {
            startButton.setStyle({ color: '#ffffff' });
        });
        startButton.on('pointerout', () => {
            startButton.setStyle({ color: '#ffec99' });
        });
    }

    selectCharacter(characterClass)
    {
        this.wsConnection.send(JSON.stringify({
            type: 'room', 
            subType: 'setAdditionalProperties', 
            data: {"class": characterClass }
        }));
    }

    highlightSelected(selectedSprite, selectedText)
    {
        // Reset all sprites
        this.characterSprites.forEach(char => {
            char.sprite.clearTint();
            char.text.setAlpha(1);
        });
        
        // Highlight selected
        selectedSprite.setTint(0xffff00);
        selectedText.setAlpha(1);
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
