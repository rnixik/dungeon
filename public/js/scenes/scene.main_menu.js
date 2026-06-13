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
    avatarUrl = null;
    roomName = 'default';

    selectedClass = null;
    selectedColor = null;
    // Whether the player has paid to unlock color selection. Intentionally not
    // persisted anywhere: the paywall must be cleared again for every new game.
    colorsUnlocked = false;

    roomPlayersListText = null;

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

        const defaultNickname = 'Player' + Math.floor(Math.random() * 1000);

        // Try to get data from Telegram Mini App
        const tgUser = window.Telegram?.WebApp?.initDataUnsafe?.user;
        if (tgUser) {
            this.avatarUrl = tgUser.photo_url || null;
            this.nickname = (tgUser.username || tgUser.first_name || defaultNickname).trim();
        } else {
            this.nickname = defaultNickname;
        }

        // limit nickname up to 10 chars
        this.nickname = this.nickname.substring(0, 10);

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
        const spacing = 128;

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

        // Scales normalized so all sprites are 80px tall
        // Knight frame: 38x40, Rogue frame: 30x34, Mage frame: 32x32
        const knightBaseScale = 1.25;   // 40 * 2.0 = 80px
        const rogueBaseScale  = 1.25;  // 34 * 2.35 ≈ 80px
        const mageBaseScale   = 1.25;   // 32 * 2.5  = 80px
        const hoverMult = 1.1;

        // Knight
        const knightSprite = this.add.sprite(centerX - spacing, startY, 'knight');
        knightSprite.setScale(knightBaseScale);
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
            knightSprite.setScale(knightBaseScale * hoverMult);
        });
        knightSprite.on('pointerout', () => {
            knightSprite.setScale(knightBaseScale);
        });

        // Rogue (Archer)
        const rogueSprite = this.add.sprite(centerX, startY, 'rogue');
        rogueSprite.setScale(rogueBaseScale);
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
            rogueSprite.setScale(rogueBaseScale * hoverMult);
        });
        rogueSprite.on('pointerout', () => {
            rogueSprite.setScale(rogueBaseScale);
        });

        // Mage
        const mageSprite = this.add.sprite(centerX + spacing, startY, 'mage');
        mageSprite.setScale(mageBaseScale);
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
            mageSprite.setScale(mageBaseScale * hoverMult);
        });
        mageSprite.on('pointerout', () => {
            mageSprite.setScale(mageBaseScale);
        });

        // Store references for highlighting
        this.characterSprites = [
            { sprite: knightSprite, text: knightText },
            { sprite: rogueSprite, text: rogueText },
            { sprite: mageSprite, text: mageText }
        ];

        this.selectionFrame = this.add.graphics();
        this.selectionFrame.lineStyle(3, 0xffec99, 1);
        this.selectionFrame.setVisible(false);

        // Color selection (behind a paywall)
        this.displayColorSelection(centerX, startY + 150);

        // Start button
        const startButton = this.make.text({
            x: centerX,
            y: startY + 290,
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
            startButton.setText('Starting...');
            startButton.disableInteractive();
            this.loadingSpinner.setPosition(centerX, startY + 360);
            this.loadingSpinner.setVisible(true);
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
        this.selectedClass = characterClass;
        this.sendPlayerProperties();
    }

    sendPlayerProperties()
    {
        const props = {};
        if (this.selectedClass) {
            props.class = this.selectedClass;
        }
        if (this.avatarUrl) {
            props.avatarUrl = this.avatarUrl;
        }
        // Only send a custom color once it has been unlocked and chosen.
        // Otherwise the server keeps assigning a random one.
        if (this.colorsUnlocked && this.selectedColor) {
            props.color = this.selectedColor;
        }
        this.wsConnection.send(JSON.stringify({
            type: 'room',
            subType: 'setAdditionalProperties',
            data: props
        }));
    }

    highlightSelected(selectedSprite, selectedText)
    {
        this.characterSprites.forEach(char => {
            char.sprite.clearTint();
        });

        const fw = 110;
        const fh = 110;
        this.selectionFrame.clear();
        this.selectionFrame.lineStyle(3, 0xffec99, 1);
        this.selectionFrame.strokeRoundedRect(
            selectedSprite.x - fw / 2,
            selectedSprite.y - fh / 2,
            fw, fh, 8
        );
        this.selectionFrame.setVisible(true);
    }

    // Palette must match playerColors in internal/game/game.go. The server
    // rejects any color that is not in this list.
    static PLAYER_COLORS = [
        '0xe74c3c', '0x3498db', '0x2ecc71', '0xf1c40f', '0x9b59b6',
        '0xe67e22', '0x1abc9c', '0xff69b4', '0xcd853f', '0x00bcd4',
        '0xff5722', '0x8bc34a', '0x673ab7', '0xff9800', '0x03a9f4',
        '0xe91e63', '0xf06292', '0x26c6da', '0xd4e157', '0xa1887f',
    ];

    displayColorSelection(centerX, y)
    {
        // Section header
        this.make.text({
            x: centerX,
            y: y,
            text: "Player Color",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '20px',
                color: '#ffffff',
                stroke: '#000000',
                strokeThickness: 3
            },
            add: true
        });

        // Swatch grid (2 rows of 10)
        const colors = MainMenu.PLAYER_COLORS;
        const cols = 10;
        const swatch = 20;
        const gap = 6;
        const cell = swatch + gap;
        const gridWidth = cols * cell - gap;
        const startX = centerX - gridWidth / 2 + swatch / 2;
        const gridY = y + 32;

        this.colorSwatches = [];
        this.colorSelectionFrame = this.add.graphics();
        this.colorSelectionFrame.setVisible(false);

        colors.forEach((colorStr, i) => {
            const col = i % cols;
            const row = Math.floor(i / cols);
            const sx = startX + col * cell;
            const sy = gridY + row * cell;

            const rect = this.add.rectangle(sx, sy, swatch, swatch, Number(colorStr));
            rect.setStrokeStyle(1, 0x000000);
            rect.setAlpha(0.3); // dimmed while locked
            this.colorSwatches.push({ rect, colorStr });
        });

        // Lock overlay button, shown until the color pack is purchased.
        this.colorLockButton = this.make.text({
            x: centerX,
            y: gridY + cell / 2,
            text: "🔒  Unlock Colors  —  $0.99",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial',
                fontSize: '20px',
                color: '#ffec99',
                backgroundColor: '#000000aa',
                padding: { x: 10, y: 6 },
                stroke: '#000000',
                strokeThickness: 2
            },
            add: true
        }).setInteractive({ useHandCursor: true });

        this.colorLockButton.on('pointerdown', () => this.showPaymentModal());
        this.colorLockButton.on('pointerover', () => this.colorLockButton.setStyle({ color: '#ffffff' }));
        this.colorLockButton.on('pointerout', () => this.colorLockButton.setStyle({ color: '#ffec99' }));
    }

    unlockColors()
    {
        this.colorsUnlocked = true;
        if (this.colorLockButton) {
            this.colorLockButton.destroy();
            this.colorLockButton = null;
        }
        this.colorSwatches.forEach(({ rect, colorStr }) => {
            rect.setAlpha(1);
            rect.setInteractive({ useHandCursor: true });
            rect.on('pointerdown', () => this.selectColor(colorStr, rect));
        });
    }

    selectColor(colorStr, rect)
    {
        this.selectedColor = colorStr;

        const fw = 26;
        const fh = 26;
        this.colorSelectionFrame.clear();
        this.colorSelectionFrame.lineStyle(3, 0xffffff, 1);
        this.colorSelectionFrame.strokeRoundedRect(
            rect.x - fw / 2,
            rect.y - fh / 2,
            fw, fh, 4
        );
        this.colorSelectionFrame.setVisible(true);

        this.sendPlayerProperties();
    }

    // Mock payment flow. No real transaction happens; this stands in for the
    // payment integration that will be added later.
    showPaymentModal()
    {
        const centerX = this.cameras.main.width / 2;
        const centerY = this.cameras.main.height / 2;
        const modal = [];

        // Click-blocking dim overlay
        const overlay = this.add.rectangle(
            centerX, centerY,
            this.cameras.main.width, this.cameras.main.height,
            0x000000, 0.6
        ).setInteractive();
        modal.push(overlay);

        const panel = this.add.rectangle(centerX, centerY, 360, 220, 0x1e1e2a)
            .setStrokeStyle(2, 0xffec99);
        modal.push(panel);

        modal.push(this.make.text({
            x: centerX, y: centerY - 75,
            text: "Premium Color Pack",
            origin: { x: 0.5, y: 0.5 },
            style: { fontFamily: 'Arial', fontSize: '22px', color: '#ffec99' },
            add: true
        }));

        modal.push(this.make.text({
            x: centerX, y: centerY - 35,
            text: "Pick your own player color\nfor this game — $0.99",
            origin: { x: 0.5, y: 0.5 },
            align: 'center',
            style: { fontFamily: 'Arial', fontSize: '15px', color: '#ffffff' },
            add: true
        }));

        const status = this.make.text({
            x: centerX, y: centerY + 5,
            text: "",
            origin: { x: 0.5, y: 0.5 },
            style: { fontFamily: 'Arial', fontSize: '14px', color: '#74c0fc' },
            add: true
        });
        modal.push(status);

        const closeModal = () => modal.forEach(o => o.destroy());

        const payButton = this.make.text({
            x: centerX - 70, y: centerY + 55,
            text: "Pay $0.99",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial', fontSize: '18px', color: '#51cf66',
                backgroundColor: '#000000', padding: { x: 12, y: 6 }
            },
            add: true
        }).setInteractive({ useHandCursor: true });
        modal.push(payButton);

        const cancelButton = this.make.text({
            x: centerX + 70, y: centerY + 55,
            text: "Cancel",
            origin: { x: 0.5, y: 0.5 },
            style: {
                fontFamily: 'Arial', fontSize: '18px', color: '#ff6b6b',
                backgroundColor: '#000000', padding: { x: 12, y: 6 }
            },
            add: true
        }).setInteractive({ useHandCursor: true });
        modal.push(cancelButton);

        cancelButton.on('pointerdown', closeModal);

        payButton.on('pointerdown', () => {
            // Mock the asynchronous payment confirmation.
            payButton.disableInteractive();
            cancelButton.disableInteractive();
            payButton.setText("Processing...").setStyle({ color: '#cccccc' });
            status.setText("Contacting payment provider (mock)...");
            this.time.delayedCall(900, () => {
                status.setText("Payment successful!").setStyle({ color: '#51cf66' });
                this.time.delayedCall(500, () => {
                    closeModal();
                    this.unlockColors();
                });
            });
        });
    }

    updateRoomPlayerList(playerList)
    {
        let playerListStr = "Players in Room:\n";
        for (const player of playerList) {
            playerListStr += "- " + player.nickname;
            if (player.id === this.myClientId) {
                playerListStr += " (You)";
            }
            playerListStr += "\n";
        }
        this.roomPlayersListText.setText(playerListStr);
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
        const spammingEvents = [
            'CreaturesPosUpdateEvent',
            'CreaturesStatsUpdateEvent',
            'TrapStateChangedEvent'
        ];
        if (json.name && !spammingEvents.includes(json.name)) {
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
        if (json.name === 'RoomJoinedEvent') {
            this.roomPlayersListText = this.make.text({
                x: 10,
                y: 10,
                text: "Players in Room:\n",
                style: {
                    fontFamily: 'Arial',
                    fontSize: '10px',
                    color: '#ffffff',
                },
                add: true
            });

            this.updateRoomPlayerList(json.data.room.members);

            return;
        }
        if (json.name === 'RoomPlayersUpdateEvent') {
            this.updateRoomPlayerList(json.data.room.members);

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
