const DEBUG = false;

// Layering (z-index)
const DEPTH_FLOOR = 0;
const DEPTH_WALLS = 5;
const DEPTH_DECOR = 8;
const DEPTH_DEAD = 10;
const DEPTH_OBJECTS = 20;
const DEPTH_PLAYER = 30;
const DEPTH_MONSTER = 40;
const DEPTH_PROJECTILES = 200;
const DEPTH_UPPER_WALLS = 500;
const DEPTH_DARKNESS = 9000;
const DEPTH_UI = 10000;

const PLAYER_SCALE = 0.8;

// Global darkness opacity (0..1)
const DARKNESS_ALPHA = 0.9;

// Player light (soft)
const LIGHT_DIAMETER    = 420;   // px
const LIGHT_FEATHER     = 100;    // px soft edge
const LIGHT_MASK_PLAYER = 'mask_player';

// Bullet glow (soft)
const BULLET_DIAMETER     = 170; // px
const BULLET_FEATHER      = 40;  // px
const LIGHT_MASK_BULLET   = 'mask_bullet';
const BULLET_SAMPLE_MS    = 40;  // position sampling interval
const BULLET_POINT_TTL_MS = 150; // lifetime of a glow point
const BULLET_TRAIL_CAP    = 256; // safety cap

// Colors (for debug raycast fill)
const BLACK = 0x000000;
const WHITE = 0xffffff;
const FILL_COLOR = BLACK;
const DEBUG_STROKE_COLOR = WHITE;
const DEBUG_FILL_COLOR = 0xff0000;

const DAMAGE_KIND_FIREBALL = 'fireball';
const DAMAGE_KIND_ARROW = 'arrow';
const DAMAGE_KIND_EXPLOSION  = 'explosion';
const DAMAGE_KIND_SPIKE = 'spike';
const DAMAGE_KIND_LIGHTNING = 'lightning';

// Phaser shortcuts
const { Circle, Line, Point, Rectangle } = Phaser.Geom;
const { EPSILON } = Phaser.Math;
const { Extend } = Line;
const { LineToLine } = Phaser.Geom.Intersects;

class Game extends Phaser.Scene {
    // runtime fields
    player;
    cursors;
    joystick;
    buttonFire;
    buttonDodge;
    buttonFs;
    _joystickThumb;
    _attackBtnDown = false;
    _dodgeBtnDown = false;

    key1;
    key2;
    key3;
    key1Collected = true;
    key2Collected = false;
    key3Collected = false;

    map;
    layerFloor;
    layerWalls;
    layerAbyss;
    layerDecor;
    projectiles;
    
    // traps
    traps = {};

    // lighting
    rt;               // RenderTexture for darkness
    lightDiameter = LIGHT_DIAMETER;
    bulletGlowTrail = [];
    _lastBulletSampleAt = 0;

    // raycast data
    mask;
    graphics;
    vertices;
    edges;
    rays;
    raycastByAreas = [];
    prevAreaId = null;

    direction = 'right';

    // UI scale helpers
    uiScaleX;
    uiScaleY;

    // server connection
    myClientId;
    myNickname;
    sendGameCommand;
    isAttacking = false;
    lastAttackTime = 0;
    isMoving = false;
    wasMoving = false;
    isDodging = false;
    lastDodgeTime = 0;
    isDead = false;
    deadText = null;
    respawnButton = null;

    lastMoveSentTime = 0;
    moveCommandInterval = 1000 / 45; // ms
    dodgeInterval = 2000;

    players = {};
    monsters = {};
    gameObjects = {};

    level = 1;
    xp;
    nextLevelXp;
    xpBar;

    inventory = [];
    currentItemIndex = 0;
    footprintGraphics = [];

    // player list
    latestPlayerStats = [];
    playerListVisible = false;
    _playerListBtn = null;
    _playerListPanel = null;
    _itemFrame = null;
    _itemArrowLeft = null;
    _itemArrowRight = null;
    _currentItemSprite = null;
    _currentItemCountText = null;
    _itemFrameX = 0;
    _itemFrameY = 0;

    constructor () {
        super({ key: 'Game' });

        Object.assign(this, GameEventHandler);
    }

    create (data) {
        const gameData = data.gameData;
        this.myClientId = gameData.playerData.clientId;
        this.myNickname = gameData.playerData.nickname;
        console.log("Game started. My client id: " + this.myClientId + ", my nickname: " + this.myNickname);
        this.sendGameCommand = data.sendGameCommand;
        const self = this;
        data.setOnIncomingGameEventCallback(function (name, data) {
            self.onIncomingGameEvent(name, data);
        });

        // viewport scale for UI placement
        this.uiScaleX = this.scale.width / 800;
        this.uiScaleY = this.scale.height / 600;

        // --- Tilemap & layers ---
        // create tiled tilemap from server map data
        this.cache.tilemap.add('map', {format: 1, data: gameData.mapData});
        this.map = this.make.tilemap({ key: 'map' });

        const tiles = this.map.addTilesetImage('catacombs', 'tiles');

        this.layerFloor = this.map.createLayer('floor', tiles, 0, 0).setDepth(DEPTH_FLOOR);
        this.layerWalls = this.map.createLayer('walls', tiles, 0, 0).setDepth(DEPTH_WALLS);
        this.layerAbyss = this.map.createLayer('abyss', tiles, 0, 0).setDepth(DEPTH_WALLS);
        this.layerDecor = this.map.createLayer('decor', tiles, 0, 0).setDepth(DEPTH_DECOR);
        this.layerWalls.setCollisionByProperty({ collides: true });
        this.layerAbyss.setCollisionByProperty({ collides: true });

        // Build an overlay layer rendered above the player.
        // For each south-facing wall face tile (extra_z), copy it and the solid wall
        // row directly below it into this layer — so the face tile covers the player's
        // mid-section and the solid wall row covers the player's legs when they stand
        // next to the lower wall.  Tiles are NOT removed from layerWalls so collision
        // is unaffected.
        const layerWallsUpper = this.map.createBlankLayer('upper_walls', tiles, 0, 0, this.map.width, this.map.height);
        this.layerWalls.forEachTile((tile) => {
            if (tile.properties.extra_z === true) {
                layerWallsUpper.putTileAt(tile.index, tile.x, tile.y);
                const tileBelow = this.layerWalls.getTileAt(tile.x, tile.y + 1);
                if (tileBelow && tileBelow.index > 0) {
                    layerWallsUpper.putTileAt(tileBelow.index, tile.x, tile.y + 1);
                }
            }
        });
        layerWallsUpper.setDepth(DEPTH_UPPER_WALLS);

        this.player = new MyPlayer(gameData.playerData.class, this, gameData.playerData);
        this.level = gameData.playerData.level;
        this.xp = gameData.playerData.xp;
        this.nextLevelXp = gameData.playerData.nextLevelXp;

        this.physics.add.collider(this.player, this.layerWalls);
        this.physics.add.collider(this.player, this.layerAbyss);

        // Camera
        this.cameras.main.setBounds(0, 0, this.map.widthInPixels, this.map.heightInPixels);
        this.cameras.main.startFollow(this.player);

        this.projectiles = new AllProjectilesGroup(this, this.layerWalls, this.onBulletHitPlayer, this.onBulletHitMonster);
        this.projectiles.addPlayer(this.player);

        // Spawn objects
        for (const i in gameData.gameObjects) {
            const o = gameData.gameObjects[i];
            const id = o.id;
            this.gameObjects[id] = GameObject.SpawnNewObject(this, o);
            this.projectiles.addObject(this.gameObjects[id]);
            this.physics.add.collider(this.player, this.gameObjects[id]);
        }

        this.key1Collected = gameData.keysCollected["1"];
        this.key2Collected = gameData.keysCollected["2"];
        this.key3Collected = gameData.keysCollected["3"];

        // Initialize traps from new system
        if (gameData.traps) {
            for (const trapData of gameData.traps) {
                this.createTrapSprite(trapData.trapId, trapData.x, trapData.y, trapData.frame);
            }
        }
        
        // Legacy spike events support
        for (const i in gameData.spikeEvents) {
            this.SpawnSpikeEvent(gameData.spikeEvents[i]);
        }
        for (const i in gameData.updateTilesEvents) {
            this.UpdateTilesEvent(gameData.updateTilesEvents[i]);
        }

        // Input
        this.cursors = this.input.keyboard.createCursorKeys();
        const spaceBar = this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.SPACE);
        spaceBar.on('down', () => this.dodge());
        const enter = this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.ENTER);
        enter.on('down', () => this.attack());

        // Darkness RT + masks
        this._initDarknessRT();
        createRadialMaskTexture(this, LIGHT_MASK_PLAYER,  LIGHT_DIAMETER,  LIGHT_FEATHER);
        createRadialMaskTexture(this, LIGHT_MASK_BULLET,  BULLET_DIAMETER, BULLET_FEATHER);

        // Resize handler: rebuild RT & reposition UI
        this.scale.on('resize', () => {
            this._initDarknessRT();
            this.addKeysIcons();
            this.updateXpBar();
            this.addMobileButtons();
            this.addItemSelector();
            this.addPlayerListButton();
            this.renderPlayerList();
        });

        // Debug polyline graphics for raycast mask
        this.graphics = this.make.graphics({ lineStyle: { color: DEBUG_STROKE_COLOR, width: 0.5 } });
        this.monsterGraphics = this.make.graphics({});
        if (DEBUG) {
            this.graphics.setAlpha(0.5);
            this.add.existing(this.graphics);
        } else {
            this.mask = new Phaser.Display.Masks.GeometryMask(this, this.graphics);
            this.monsterMask = new Phaser.Display.Masks.GeometryMask(this, this.monsterGraphics);
        }

        // Apply (optional) geometric mask to world layers/actors (not UI)
        this.layerFloor.setMask(this.mask);

        // --- Build occluder rectangles from wall tiles ---
        const rectsByAreas = getCollisionRectsFromMapData(gameData.mapData);
        for (const area of rectsByAreas) {
            this.raycastByAreas.push({
                id: area.id,
                center: area.center,
                edges: area.rects.flatMap(getRectEdges),
                vertices: area.rects.flatMap(getRectVertices),
                rays: area.rects.flatMap(getRectVertices).map(() => new Line()),
                rects: area.rects
            });
        }

        // UI
        this.addKeysIcons();
        this.updateXpBar();
        this.input.addPointer(2); // allow 3 simultaneous pointers (joystick + 2 buttons)
        this.addMobileButtons();
        this.inventory = gameData.inventory || [];
        this.player.speedBoostPercent = gameData.speedBoostPercent || 0;
        this.addItemSelector();
        this.addPlayerListButton();
    }

    update (time, delta) {
        // Movement
        const joy = this.joystick?.createCursorKeys?.() || {left:{isDown:false},right:{isDown:false},up:{isDown:false},down:{isDown:false}};

        // Web slow: check if player overlaps any active web area
        const now = Date.now();
        let inWeb = false;
        if (this.activeWebs) {
            for (const web of this.activeWebs) {
                if (web.expiresAt > now &&
                    Math.abs(this.player.x - web.x) <= web.halfSize &&
                    Math.abs(this.player.y - web.y) <= web.halfSize) {
                    inWeb = true;
                    break;
                }
            }
        }
        this.player.webSlowMultiplier = inWeb ? 0.3 : 1;

        // Jelly aura slow: 20% speed reduction within 3 tiles of any alive jelly
        const JELLY_AURA_RADIUS = 48;
        let inJellyAura = false;
        for (const id in this.monsters) {
            const m = this.monsters[id];
            if ((m.kind === 'jelly' || m.kind === 'jelly_small') && !m.isCorpse) {
                const dx = this.player.x - m.x;
                const dy = this.player.y - m.y;
                if (dx * dx + dy * dy <= JELLY_AURA_RADIUS * JELLY_AURA_RADIUS) {
                    inJellyAura = true;
                    break;
                }
            }
        }
        this.player.jellyAuraSlow = inJellyAura;

        this.player.body.setVelocity(0);
        if (this.isDodging) {
            const velocity = 300;
            const vector = direction4xToVector(this.direction);
            this.player.body.setVelocityX(vector.x * velocity);
            this.player.body.setVelocityY(vector.y * velocity);
        } else {
            const velocity = this.player.getMovementVelocity();
            this.player.isAttacking = this.isAttacking;

            if (this.cursors.left.isDown || joy.left.isDown)  this.player.body.setVelocityX(-velocity);
            else if (this.cursors.right.isDown || joy.right.isDown) this.player.body.setVelocityX(velocity);

            if (this.cursors.up.isDown || joy.up.isDown)      this.player.body.setVelocityY(-velocity);
            else if (this.cursors.down.isDown || joy.down.isDown) this.player.body.setVelocityY(velocity);

            // do not change the direction while dodging
            if (this.cursors.left.isDown || joy.left.isDown) {
                this.direction='left';
                this.isMoving = true;
            } else if (this.cursors.right.isDown || joy.right.isDown) {
                this.direction='right';
                this.isMoving = true;
            } else if (this.cursors.up.isDown || joy.up.isDown) {
                this.direction='up';
                this.isMoving = true;
            } else if (this.cursors.down.isDown || joy.down.isDown) {
                this.direction='down';
                this.isMoving = true;
            } else {
                this.isMoving = false;
            }
        }

        if (this.isDodging) {
            // TODO: replace with dodge animation
            this.player.playIdleAnimation(this.direction);
        } else if (this.isAttacking) {
            this.player.playAttackAnimation(this.direction);
        } else if (this.isMoving) {
            this.player.playMoveAnimation(this.direction);
        } else {
            this.player.playIdleAnimation(this.direction);
        }

        // Mobile button visual states
        this.updateButtonStates();

        // Lighting (spotlights + bullet glow)
        this.updateMaskLight();

        // Raycast dynamic shadows
        this.updateMaskRaycast();

        if ((this.isMoving || this.isDodging) && this.lastMoveSentTime + this.moveCommandInterval < time) {
            this.sendGameCommand('PlayerMoveCommand', {
                x: Math.round(this.player.x),
                y: Math.round(this.player.y),
                direction: this.direction,
                isMoving: this.isMoving
            });
            this.lastMoveSentTime = time;
        } else if (this.wasMoving && !this.isMoving && !this.isDodging) {
            this.sendGameCommand('PlayerMoveCommand', {
                x: Math.round(this.player.x),
                y: Math.round(this.player.y),
                direction: this.direction,
                isMoving: false
            });
        }
        this.wasMoving = this.isMoving;
    }

    onIncomingGameEvent (name, data) {
        if (typeof this[name] === 'function') {
            this[name](data);
            return;
        }

        console.log('INCOMING GAME EVENT', name, data);
    }

    updatePlayerPos(p)
    {
        if (p.clientId === this.myClientId) {
            return;
        }

        if (!this.players[p.clientId]) {
            return;
        }

        this.players[p.clientId].updatePosition(p);
    }

    updateMonsterPos(m)
    {
        if (!this.monsters[m.id]) {
            return;
        }

        this.monsters[m.id].updatePosition(m);
    }

    attack() {
        switch (this.player.kind) {
            case 'mage':
                this.fireballAttack();
                return;
            case 'knight':
                this.swordAttack();
                return;
            case 'rogue':
                this.shotArrow();
        }
    }

    dodge() {
        if (this.isDodging) {
            return;
        }
        if (this.isAttacking) {
            return;
        }
        if (this.time.now - this.lastDodgeTime < this.dodgeInterval) {
            return;
        }

        this.lastDodgeTime = this.time.now;
        this.isDodging = true;

        this.time.delayedCall(400, () => {this.isDodging = false;}, [], this);

        this.sendGameCommand('DodgeCommand', {
            x: Math.round(this.player.x),
            y: Math.round(this.player.y),
            direction: this.direction,
        });
    }

    fireballAttack() {
        this.isAttacking = true;
        this.time.delayedCall(500, () => {this.isAttacking = false;}, [], this);

        this.sendGameCommand('CastFireballCommand', {
            x: Math.round(this.player.x),
            y: Math.round(this.player.y),
            direction: this.direction,
        });
    }

    swordAttack() {
        if (this.isAttacking) {
            return;
        }
        const attackTs = this.time.now;
        this.lastAttackTime = attackTs;
        this.isAttacking = true;

        this.time.delayedCall(1000, () => {if (attackTs === this.lastAttackTime) { this.isAttacking = false; } }, [], this);

        this.sendGameCommand('SwordAttackCommand', {
            x: Math.round(this.player.x),
            y: Math.round(this.player.y),
            direction: this.direction,
        });
    }

    shotArrow() {
        if (this.isAttacking) {
            return;
        }
        const attackTs = this.time.now;
        this.lastAttackTime = attackTs;
        this.isAttacking = true;

        this.time.delayedCall(500, () => {if (attackTs === this.lastAttackTime) { this.isAttacking = false; } }, [], this);

        this.sendGameCommand('ShootArrowCommand', {
            x: Math.round(this.player.x),
            y: Math.round(this.player.y),
            direction: this.direction,
        });
    }

    onBulletHitPlayer(bullet, player, kind)
    {
        if (DEBUG) console.log('hit player', bullet.clientId, player.id, kind);
        if (bullet.monsterId && player.id === this.myClientId) {
            // hit caused by monster's bullet on ourselves
            this.sendGameCommand('HitPlayerCommand', {
                monsterId: bullet.monsterId,
                targetClientId: this.myClientId,
                kind: kind,
            });
            return;
        }

        if (bullet.clientId !== this.myClientId) {
            return; // only report hits caused by our own bullets
        }
        this.sendGameCommand('HitPlayerCommand', {
            originClientId: bullet.clientId,
            targetClientId: player.id,
            kind: kind,
        });
    }

    onBulletHitMonster(bullet, monster, kind)
    {
        if (DEBUG) console.log('hit monster', bullet.clientId, this.myClientId, bullet.monsterId, monster.id, kind);
        if (bullet.monsterId) {
            return; // ignore hitting monsters by monster bullets
        }

        if (bullet.clientId !== this.myClientId) {
            return; // only report hits caused by our own bullets
        }
        this.sendGameCommand('HitMonsterCommand', {
            originClientId: bullet.clientId,
            monsterId: monster.id,
            kind: kind,
        });
        if (DEBUG) console.log('monster hit command sent');
    }

    // --- Darkness RenderTexture setup ---
    _initDarknessRT() {
        if (this.rt) this.rt.destroy();
        this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
        this.rt.setOrigin(0, 0);
        this.rt.setScrollFactor(0, 0);
        this.rt.setDepth(DEPTH_DARKNESS);
        this.rt.setAlpha(DARKNESS_ALPHA);
    }

    // --- Lighting pass: fill darkness, erase soft masks at actors & bullet glow ---
    updateMaskLight () {
        const cam = this.cameras.main;
        this.rt.clear();
        this.rt.fill(0x000000, 1);

        if (this.isDead) {
            this.rt.setAlpha(1)

            return;
        }

        const eraseAt = (key, x, y, diameter) => {
            const h = diameter / 2;
            this.rt.erase(key, (x - h) - cam.scrollX, (y - h) - cam.scrollY);
        };

        eraseAt(LIGHT_MASK_PLAYER, this.player.x,     this.player.y,     LIGHT_DIAMETER);

        if (!this.projectiles) {
            return;
        }

        // Sample bullets periodically to build a short-lived glow trail
        const now = this.time.now;
        if (now - this._lastBulletSampleAt >= BULLET_SAMPLE_MS) {
            this._lastBulletSampleAt = now;
            for (const s of this.projectiles.getAllIlluminatedSprites()) {
                this._pushBulletLight(s.x, s.y, now + BULLET_POINT_TTL_MS);
            }
        }

        for (let i = this.bulletGlowTrail.length - 1; i >= 0; i--) {
            const p = this.bulletGlowTrail[i];
            if (p.expiresAt <= now) { this.bulletGlowTrail.splice(i, 1); continue; }
            eraseAt(LIGHT_MASK_BULLET, p.x, p.y, BULLET_DIAMETER);
        }
    }

    // --- Raycast visibility mask (draws into this.graphics used as GeometryMask) ---
    updateMaskRaycast () {
        if (this.isDead) {
            return;
        }

        // find closest area(s) to player and raycast there
        for (const area of this.raycastByAreas) {
            if (Math.abs(this.player.x - area.center.x) < 400 && Math.abs(this.player.y - area.center.y) < 300) {
                if (this.prevAreaId !== area.id) {
                    console.log("Switched to raycast area id:", area.id, "rects num:", area.rects.length);
                    this.prevAreaId = area.id;

                    if (DEBUG) {
                        const rectGraphics = this.add.graphics({ fillStyle: { color: 0x0000aa } }).setDepth(DEPTH_UI - 1);
                        for (const r of area.rects) rectGraphics.fillRectShape(r);
                        const rectVertGraphics = this.add.graphics({ fillStyle: { color: 0x00aaaa } }).setDepth(DEPTH_UI - 1);
                        for (const r of area.rects) for (const v of getRectVertices(r)) rectVertGraphics.fillPointShape(v, 4);
                    }
                }

                const vertices = area.vertices;
                const edges    = area.edges;
                const rays     = area.rays;

                const vis = calc(this.player, vertices, edges, rays);
                draw(this.graphics, vis, rays, edges);
                drawMonsterMask(this.monsterGraphics, vis, this.player.y);

                return;
            }
        }
    }

    addKeysIcons() {
        this.addKey(this.key1, this.key1Collected, 1);
        this.addKey(this.key2, this.key2Collected, 2);
        this.addKey(this.key3, this.key3Collected, 3);
    }

    addKey(spriteProp, isCollectedProp, number) {
        if (spriteProp) spriteProp.destroy(true, true);

        const btnScale = Math.max(this.uiScaleX, this.uiScaleY);

        const offsetX = (number - 1) * 20;
        spriteProp = this.add.sprite(40 * this.uiScaleX + offsetX, 40 * this.uiScaleY, 'key').anims.play('key');
        spriteProp.setScrollFactor(0, 0).setScale(btnScale).setDepth(DEPTH_UI);

        if (isCollectedProp) {
            spriteProp.setTint(0xffffff);
        } else {
            spriteProp.setTint(0x000000);
        }
    }

    updateXpBar() {
        if (this.xpBar) {
            this.xpBar.destroy(true, true);
        }

        const barWidth = 200 * this.uiScaleX;
        const barHeight = 20 * this.uiScaleY;
        const x = (this.scale.width - barWidth) / 2;
        const y = this.scale.height - 40 * this.uiScaleY;

        this.xpBar = this.add.graphics();
        this.xpBar.setScrollFactor(0, 0).setDepth(DEPTH_UI);
        this.xpBar.fillStyle(0xffffff, 0.5);
        this.xpBar.fillRect(x, y, barWidth, barHeight);

        const xpRatio = Phaser.Math.Clamp(this.xp / this.nextLevelXp, 0, 1);
        this.xpBar.fillStyle(0xffffff, 1);
        this.xpBar.fillRect(x + 2, y + 2, (barWidth - 4) * xpRatio, barHeight - 4);
    }

    addMobileButtons () {
        if (this.joystick) this.joystick.destroy(true, true);
        if (this.buttonFire) this.buttonFire.destroy(true, true);
        if (this.buttonDodge) this.buttonDodge.destroy(true, true);
        if (this.buttonFs) this.buttonFs.destroy(true, true);

        const btnScale = Math.max(this.uiScaleX, this.uiScaleY);
        const joyX = 90 * this.uiScaleX;
        const joyY = this.scale.height - 90 * this.uiScaleY;

        const joyBase = this.add.image(0, 0, 'ui', 'joystick_dark_active')
            .setDepth(DEPTH_UI).setScale(btnScale).setAlpha(0.8).setScrollFactor(0, 0);
        this._joystickThumb = this.add.image(0, 0, 'ui', 'joystick_center_dot')
            .setDepth(DEPTH_UI + 1).setScale(btnScale).setAlpha(0.9).setScrollFactor(0, 0);

        this.joystick = this.plugins.get('rexvirtualjoystickplugin').add(this, {
            x: joyX,
            y: joyY,
            radius: 70 * btnScale,
            base: joyBase,
            thumb: this._joystickThumb,
            dir: '8dir'
        });

        this.buttonFire = this.add.image(
            this.scale.width - 85 * this.uiScaleX,
            this.scale.height - 85 * this.uiScaleY,
            'ui', 'button_A_red_active'
        );
        this.buttonFire.setScrollFactor(0, 0).setScale(btnScale).setAlpha(0.85)
            .setInteractive({ useHandCursor: true }).setDepth(DEPTH_UI);
        this.buttonFire.on('pointerdown', () => { this._attackBtnDown = true; this.attack(); });
        this.buttonFire.on('pointerup',   () => { this._attackBtnDown = false; });
        this.buttonFire.on('pointerout',  () => { this._attackBtnDown = false; });

        const isPortrait = this.scale.height > this.scale.width;
        this.buttonDodge = this.add.image(
            this.scale.width - 199 * (isPortrait ? (this.uiScaleX + this.uiScaleY) / 2 : this.uiScaleX),
            this.scale.height - 85 * this.uiScaleY,
            'ui', 'button_D_blue_active'
        );
        this.buttonDodge.setScrollFactor(0, 0).setScale(btnScale).setAlpha(0.85)
            .setInteractive({ useHandCursor: true }).setDepth(DEPTH_UI);
        this.buttonDodge.on('pointerdown', () => { this._dodgeBtnDown = true; this.dodge(); });
        this.buttonDodge.on('pointerup',   () => { this._dodgeBtnDown = false; });
        this.buttonDodge.on('pointerout',  () => { this._dodgeBtnDown = false; });
    }

    updateButtonStates () {
        if (!this.buttonFire || !this.buttonDodge) return;

        if (this.isAttacking) {
            this.buttonFire.setFrame('button_A_gray_inactive');
        } else if (this._attackBtnDown) {
            this.buttonFire.setFrame('button_A_gold_active');
        } else {
            this.buttonFire.setFrame('button_A_red_active');
        }

        const dodgeOnCooldown = this.time.now - this.lastDodgeTime < this.dodgeInterval;
        if (dodgeOnCooldown) {
            this.buttonDodge.setFrame('button_D_gray_inactive');
        } else if (this._dodgeBtnDown) {
            this.buttonDodge.setFrame('button_D_steel_active');
        } else {
            this.buttonDodge.setFrame('button_D_blue_active');
        }
    }

    _pushBulletLight(x, y, expiresAt) {
        this.bulletGlowTrail.push({ x, y, expiresAt });
        if (this.bulletGlowTrail.length > BULLET_TRAIL_CAP) {
            this.bulletGlowTrail.splice(0, this.bulletGlowTrail.length - BULLET_TRAIL_CAP);
        }
    }

    addItemSelector() {
        if (this._itemFrame) this._itemFrame.destroy();
        if (this._itemArrowLeft) this._itemArrowLeft.destroy();
        if (this._itemArrowRight) this._itemArrowRight.destroy();
        if (this._currentItemSprite) { this._currentItemSprite.destroy(); this._currentItemSprite = null; }
        if (this._currentItemCountText) { this._currentItemCountText.destroy(); this._currentItemCountText = null; }

        const invScale = Math.max(this.uiScaleX, this.uiScaleY) * 0.7;
        const margin = 10;

        const frameW = 80;
        const arrowLW = 42;
        const arrowRW = 40;
        const frameH = 81;

        const rightEdge = this.scale.width - margin;
        const centerY = margin + (frameH * invScale) / 2;

        const arrowRightX = rightEdge - (arrowRW * invScale) / 2;
        const frameX = arrowRightX - (arrowRW * invScale) / 2 - (frameW * invScale) / 2;
        const arrowLeftX = frameX - (frameW * invScale) / 2 - (arrowLW * invScale) / 2;

        this._itemFrameX = frameX;
        this._itemFrameY = centerY;

        this._itemFrame = this.add.image(frameX, centerY, 'ui', 'item_frame')
            .setScrollFactor(0, 0)
            .setScale(invScale)
            .setDepth(DEPTH_UI)
            .setInteractive({ useHandCursor: true });
        this._itemFrame.on('pointerdown', () => this.useCurrentItem());

        this._itemArrowLeft = this.add.image(arrowLeftX, centerY, 'ui', 'arrow_left_brown')
            .setScrollFactor(0, 0)
            .setScale(invScale)
            .setDepth(DEPTH_UI)
            .setInteractive({ useHandCursor: true });
        this._itemArrowLeft.on('pointerdown', () => this.selectPrevItem());

        this._itemArrowRight = this.add.image(arrowRightX, centerY, 'ui', 'arrow_right_brown')
            .setScrollFactor(0, 0)
            .setScale(invScale)
            .setDepth(DEPTH_UI)
            .setInteractive({ useHandCursor: true });
        this._itemArrowRight.on('pointerdown', () => this.selectNextItem());

        this._drawCurrentItemIcon();
    }

    _drawCurrentItemIcon() {
        if (this._currentItemSprite) { this._currentItemSprite.destroy(); this._currentItemSprite = null; }
        if (this._currentItemCountText) { this._currentItemCountText.destroy(); this._currentItemCountText = null; }

        if (!this.inventory || this.inventory.length === 0) return;

        const invScale = Math.max(this.uiScaleX, this.uiScaleY) * 0.7;
        const item = this.inventory[this.currentItemIndex];
        if (!item) return;

        const x = this._itemFrameX;
        const y = this._itemFrameY;
        const remainingMs = item.cooldownEndTime ? Math.max(0, item.cooldownEndTime - Date.now()) : 0;
        const isOnCooldown = remainingMs > 0;
        const alpha = (item.count > 0 && !isOnCooldown) ? 1 : 0.4;

        switch (item.kind) {
            case 'healing_potion':
                this._currentItemSprite = this.add.sprite(x, y, 'potion_hp')
                    .setScrollFactor(0, 0)
                    .setScale(2 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
            case 'spikes':
                this._currentItemSprite = this.add.sprite(x, y, 'spikes', 0)
                    .setScrollFactor(0, 0)
                    .setScale(1.5 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
            case 'scroll_of_footprints':
                this._currentItemSprite = this.add.image(x, y, 'scroll_of_footprints')
                    .setScrollFactor(0, 0)
                    .setScale(2 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
            case 'scroll_of_xp':
                this._currentItemSprite = this.add.image(x, y, 'scroll_of_xp')
                    .setScrollFactor(0, 0)
                    .setScale(2 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
            case 'boots_of_haste':
                this._currentItemSprite = this.add.image(x, y, 'boots_of_haste')
                    .setScrollFactor(0, 0)
                    .setScale(2 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
            case 'scroll_of_protection':
                this._currentItemSprite = this.add.image(x, y, 'scroll_of_protection')
                    .setScrollFactor(0, 0)
                    .setScale(2 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
            case 'cloak_of_invisibility':
                this._currentItemSprite = this.add.image(x, y, 'cloak_of_invisibility')
                    .setScrollFactor(0, 0)
                    .setScale(2 * invScale)
                    .setDepth(DEPTH_UI + 1)
                    .setAlpha(alpha);
                break;
        }

        const frameW = 80;
        const frameH = 81;
        const countX = x + (frameW * invScale) / 2 - 10;
        const countY = y + (frameH * invScale) / 2 - 10;

        let countLabel;
        if (item.kind === 'cloak_of_invisibility') {
            if (isOnCooldown) {
                const secs = Math.ceil(remainingMs / 1000);
                const m = Math.floor(secs / 60);
                const s = secs % 60;
                countLabel = m > 0 ? `${m}:${String(s).padStart(2, '0')}` : `${s}s`;
            } else {
                countLabel = 'RDY';
            }
        } else {
            countLabel = `${item.count}`;
        }

        this._currentItemCountText = this.add.text(countX, countY, countLabel, {
            font: `${Math.max(10, Math.round(14 * invScale))}px Arial`,
            fill: (item.count > 0 && !isOnCooldown) ? '#ffffff' : '#888888',
            stroke: '#000000',
            strokeThickness: 2
        })
            .setScrollFactor(0, 0)
            .setDepth(DEPTH_UI + 2)
            .setOrigin(1, 1);
    }

    updateItemSelector() {
        this._drawCurrentItemIcon();
        this._restartCooldownTicker();
    }

    _restartCooldownTicker() {
        if (this._cooldownTickEvent) {
            this._cooldownTickEvent.remove(false);
            this._cooldownTickEvent = null;
        }
        const item = this.inventory && this.inventory[this.currentItemIndex];
        if (!item || !item.cooldownEndTime) return;
        const remaining = item.cooldownEndTime - Date.now();
        if (remaining <= 0) return;
        this._cooldownTickEvent = this.time.addEvent({
            delay: 1000,
            repeat: Math.ceil(remaining / 1000),
            callback: () => {
                this._drawCurrentItemIcon();
                const cur = this.inventory && this.inventory[this.currentItemIndex];
                if (!cur || !cur.cooldownEndTime || Date.now() >= cur.cooldownEndTime) {
                    this._cooldownTickEvent.remove(false);
                    this._cooldownTickEvent = null;
                }
            }
        });
    }

    selectPrevItem() {
        if (!this.inventory || this.inventory.length === 0) return;
        this.currentItemIndex = (this.currentItemIndex - 1 + this.inventory.length) % this.inventory.length;
        this._drawCurrentItemIcon();
        this._restartCooldownTicker();
    }

    selectNextItem() {
        if (!this.inventory || this.inventory.length === 0) return;
        this.currentItemIndex = (this.currentItemIndex + 1) % this.inventory.length;
        this._drawCurrentItemIcon();
        this._restartCooldownTicker();
    }

    useCurrentItem() {
        if (!this.inventory || this.inventory.length === 0 || this.isDead) return;
        const item = this.inventory[this.currentItemIndex];
        if (!item || item.count <= 0) return;
        this.sendGameCommand('UseItemCommand', { kind: item.kind });
    }

    addPlayerListButton() {
        if (this._playerListBtn) this._playerListBtn.destroy();

        const fontSize = Math.max(12, Math.round(14 * Math.max(this.uiScaleX, this.uiScaleY)));
        const count = this.latestPlayerStats.length;
        this._playerListBtn = this.add.text(this.scale.width / 2, 10 * this.uiScaleY, `Players (${count})`, {
            font: `${fontSize}px Arial`,
            fill: '#ffffff',
            backgroundColor: 'rgba(0,0,0,0.6)',
            padding: { x: 8, y: 4 }
        })
            .setOrigin(0.5, 0)
            .setScrollFactor(0, 0)
            .setDepth(DEPTH_UI)
            .setInteractive({ useHandCursor: true });
        this._playerListBtn.on('pointerdown', () => this.togglePlayerList());
    }

    togglePlayerList() {
        this.playerListVisible = !this.playerListVisible;
        this.renderPlayerList();
    }

    renderPlayerList() {
        if (this._playerListPanel) { this._playerListPanel.destroy(); this._playerListPanel = null; }
        if (!this.playerListVisible) return;

        const players = this.latestPlayerStats.slice().sort((a, b) => (b.level - a.level) || (b.hp - a.hp));
        const scale = Math.max(this.uiScaleX, this.uiScaleY);
        const fontSize = Math.max(11, Math.round(13 * scale));
        const rowH = fontSize + 9;
        const pad = 10 * scale;
        const swatch = fontSize;
        const panelW = 270 * scale;
        const panelH = pad * 2 + Math.max(1, players.length) * rowH;

        const panelX = this.scale.width / 2 - panelW / 2;
        const panelY = (10 * this.uiScaleY) + (this._playerListBtn ? this._playerListBtn.height : 0) + 6;

        const container = this.add.container(panelX, panelY).setScrollFactor(0, 0).setDepth(DEPTH_UI);

        const bg = this.add.graphics();
        bg.fillStyle(0x000000, 0.7);
        bg.fillRect(0, 0, panelW, panelH);
        bg.lineStyle(1, 0xffffff, 0.3);
        bg.strokeRect(0, 0, panelW, panelH);
        container.add(bg);

        if (players.length === 0) {
            container.add(this.add.text(pad, pad, 'No players', { font: `${fontSize}px Arial`, fill: '#aaaaaa' }));
        }

        players.forEach((p, i) => {
            const y = pad + i * rowH;
            const isMe = p.clientId === this.myClientId;
            const isDead = p.hp <= 0;

            const colorInt = parseInt((p.color || '#ffffff').replace('#', ''), 16);
            const sw = this.add.graphics();
            sw.fillStyle(colorInt, isDead ? 0.3 : 1);
            sw.fillRect(pad, y + 2, swatch, swatch);
            sw.lineStyle(1, 0xffffff, 0.4);
            sw.strokeRect(pad, y + 2, swatch, swatch);
            container.add(sw);

            const name = (p.nickname || '???') + (isMe ? ' (you)' : '');
            container.add(this.add.text(pad + swatch + 6, y, name, {
                font: `${fontSize}px Arial`,
                fill: isMe ? '#ffe066' : '#ffffff'
            }));

            const info = isDead ? `Lv.${p.level} ${p.class} · dead` : `Lv.${p.level} ${p.class} · ${p.hp}/${p.maxHp}`;
            container.add(this.add.text(panelW - pad, y, info, {
                font: `${fontSize}px Arial`,
                fill: isDead ? '#ff6666' : '#cccccc'
            }).setOrigin(1, 0));
        });

        this._playerListPanel = container;
    }
}

// --- Scene export/usage ---
var sceneConfigGame = new Game();

// ===================== Utils: Lights =====================
function createRadialMaskTexture(scene, key, diameter, feather) {
    if (scene.textures.exists(key)) return;
    const d = Math.max(8, Math.floor(diameter));
    const f = Math.max(0, Math.floor(feather));
    const tex = scene.textures.createCanvas(key, d, d);
    const ctx = tex.getContext();
    const R  = d / 2;
    const inner = Math.max(0, R - f);
    const grad = ctx.createRadialGradient(R, R, inner, R, R, R);
    grad.addColorStop(0, 'rgba(255,255,255,1)');
    grad.addColorStop(1, 'rgba(255,255,255,0)');
    ctx.fillStyle = grad;
    ctx.fillRect(0, 0, d, d);
    tex.refresh();
}

// ===================== Utils: Raycast mask =====================
function draw (graphics, vertices, rays, edges) {
    if (!vertices || vertices.length < 3) { graphics.clear(); return; }
    graphics.clear().fillStyle(FILL_COLOR).fillPoints(vertices, true);
    if (DEBUG) {
        for (const ray of rays)  graphics.strokeLineShape(ray);
        for (const edge of edges) graphics.strokeLineShape(edge);
        graphics.fillStyle(DEBUG_FILL_COLOR);
        for (const vert of vertices) graphics.fillPointShape(vert, 4);
    }
}

// Monster mask: same polygon but horizontal edges above the player are pushed up
// so sprites are not clipped at north wall faces. Vertical edges (pillars, corners)
// are left unchanged, keeping correct occlusion there.
const MONSTER_MASK_NORTH_OFFSET = 40;
function drawMonsterMask (graphics, vertices, playerY) {
    if (!vertices || vertices.length < 3) { graphics.clear(); return; }
    const n = vertices.length;
    const shifts = new Array(n).fill(0);
    for (let i = 0; i < n; i++) {
        const a = vertices[i], b = vertices[(i + 1) % n];
        if ((a.y + b.y) * 0.5 < playerY && Math.abs(a.y - b.y) < 8) {
            shifts[i] = MONSTER_MASK_NORTH_OFFSET;
            shifts[(i + 1) % n] = MONSTER_MASK_NORTH_OFFSET;
        }
    }
    const expanded = vertices.map((v, i) => shifts[i] ? { x: v.x, y: v.y - shifts[i] } : v);
    graphics.clear().fillStyle(FILL_COLOR).fillPoints(expanded, true);
}

function calc (source, vertices, edges, rays) {
    const sx = source.x, sy = source.y;
    return sortClockwise(
        rays.map((ray, i) => {
            ray.setTo(sx, sy, vertices[i].x, vertices[i].y);
            Extend(ray, 0, 1000);
            for (const edge of edges) getRayToEdge(ray, edge, _rayEdgeScratch);
            return ray.getPointB();
        }),
        source
    );
}

function getRectEdges (rect) { return [rect.getLineA(), rect.getLineB(), rect.getLineC(), rect.getLineD()]; }

function getRectVertices (rect) {
    const { left, top, right, bottom } = rect;
    const left1 = left + EPSILON,  top1 = top + EPSILON,  right1 = right - EPSILON,  bottom1 = bottom - EPSILON;
    const left2 = left - EPSILON,  top2 = top - EPSILON,  right2 = right + EPSILON,  bottom2 = bottom + EPSILON;
    return [
        new Point(left1,  top1),    new Point(right1, top1),
        new Point(right1, bottom1), new Point(left1,  bottom1),
        new Point(left2,  top2),    new Point(right2, top2),
        new Point(right2, bottom2), new Point(left2,  bottom2)
    ];
}

// Shared scratch point reused across every ray/edge intersection test so the
// per-frame raycast doesn't allocate a Point for each edge of each ray.
const _rayEdgeScratch = new Point();

function getRayToEdge (ray, edge, out) {
    if (!out) out = new Point();
    if (LineToLine(ray, edge, out)) { ray.x2 = out.x; ray.y2 = out.y; return out; }
    return null;
}

function sortClockwise (points, center) {
    const cx = center.x, cy = center.y;
    return points.sort((a, b) => {
        if (a.x - cx >= 0 && b.x - cx < 0) return -1;
        if (a.x - cx < 0 && b.x - cx >= 0) return 1;
        if (a.x - cx === 0 && b.x - cx === 0) {
            if (a.y - cy >= 0 || b.y - cy >= 0) return (a.y > b.y) ? 1 : -1;
            return (b.y > a.y) ? 1 : -1;
        }
        const det = (a.x - cx) * -(b.y - cy) - (b.x - cx) * -(a.y - cy);
        if (det < 0) return -1; if (det > 0) return 1;
        const d1 = (a.x - cx) ** 2 + (a.y - cy) ** 2;
        const d2 = (b.x - cx) ** 2 + (b.y - cy) ** 2;
        return (d1 > d2) ? -1 : 1;
    });
}

function getCollisionRectsFromMapData(mapData) {
    const rectsByAreas = [];
    const areasLayer = mapData.layers.find(l => l.name === 'area-centers');
    for (const a of areasLayer.objects) {
        rectsByAreas.push({
            "id": a.id,
            "center": new Point(a.x, a.y),
            rects: []
        });
    }

    const rectsLayer = mapData.layers.find(l => l.name === 'collision-rects');
    for (const r of rectsLayer.objects) {
        const propertiesMap = {};
        for (const p of r.properties) {
            propertiesMap[p.name] = p.value;
        }
        for (const area of rectsByAreas) {
            const aid = area.id;
            if (propertiesMap["area_" + aid]) {
                area.rects.push(new Rectangle(r.x, r.y, r.width, r.height));
            }
        }
    }

    return rectsByAreas;
}

function direction4xToVector(direction4x) {
    switch (direction4x) {
        case 'left': return new Phaser.Math.Vector2(-1, 0);
        case 'right': return new Phaser.Math.Vector2(1, 0);
        case 'up': return new Phaser.Math.Vector2(0, -1);
        case 'down': return new Phaser.Math.Vector2(0, 1);
    }
}
