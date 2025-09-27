const DEBUG = false;

// Layering (z-index)
const DEPTH_DEAD = 25;
const DEPTH_PLAYER   = 30;
const DEPTH_MONSTER   = 40;
const DEPTH_DARKNESS = 9000;
const DEPTH_UI       = 10000;

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
    map;
    layerWalls;
    layerFloor;
    bullets; // external bullets manager passed/constructed elsewhere in your code

    // lighting
    rt;               // RenderTexture for darkness
    lightDiameter = LIGHT_DIAMETER;
    bulletGlowTrail = [];
    _lastBulletSampleAt = 0;

    // raycast data
    graphics;
    vertices;
    edges;
    rays;
    direction = 'right';

    // UI scale helpers
    uiScaleX;
    uiScaleY;

    // server connection
    myClientId;
    myNickname;
    sendGameCommand;
    isMoving = false;
    isDead = false;

    lastMoveSentTime = 0;
    moveCommandInterval = 1000 / 60; // ms

    players = {};
    monsters = {};

    constructor () {
        super({ key: 'Game' });
    }

    create (data) {
        this.myClientId = data.myClientId;
        this.myNickname = data.myNickname;
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
        this.map = this.make.tilemap({ key: 'map' });
        const tiles = this.map.addTilesetImage('environment', 'tiles');

        this.layerFloor = this.map.createLayer('floor', tiles, 0, 0);
        this.layerWalls = this.map.createLayer('walls', tiles, 0, 0);
        this.layerWalls.setCollisionByProperty({ collides: true });

        // --- Sprites ---
        this.player = this.physics.add.sprite(120, 140, 'player', 1).setScale(2).setDepth(DEPTH_PLAYER);
        this.physics.add.collider(this.player, this.layerWalls);
        this.player.hp = 100;
        this.player.hpText = this.add.text(0, 0, '100/100', { font: '8px Arial', fill: '#ffffff' }).setOrigin(0.5, 1).setDepth(DEPTH_PLAYER + 1);

        // Camera
        this.cameras.main.setBounds(0, 0, this.map.widthInPixels, this.map.heightInPixels);
        this.cameras.main.startFollow(this.player);

        // Bullets manager (assumes you have a Bullets class)
        this.bullets = new Bullets(this, this.layerWalls, this.onBulletHitPlayer, this.onBulletHitMonster);

        // Input
        this.cursors = this.input.keyboard.createCursorKeys();
        const spaceBar = this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.SPACE);
        spaceBar.on('down', () => this.castFireball());

        // Darkness RT + masks
        this._initDarknessRT();
        createRadialMaskTexture(this, LIGHT_MASK_PLAYER,  LIGHT_DIAMETER,  LIGHT_FEATHER);
        createRadialMaskTexture(this, LIGHT_MASK_BULLET,  BULLET_DIAMETER, BULLET_FEATHER);

        // Resize handler: rebuild RT & reposition UI
        this.scale.on('resize', () => {
            this._initDarknessRT();
            this.addMobileButtons();
        });

        // Debug polyline graphics for raycast mask
        this.graphics = this.make.graphics({ lineStyle: { color: DEBUG_STROKE_COLOR, width: 0.5 } });
        let mask;
        if (DEBUG) {
            mask = null;
            this.graphics.setAlpha(0.5);
            this.add.existing(this.graphics);
        } else {
            mask = new Phaser.Display.Masks.GeometryMask(this, this.graphics);
        }

        // Apply (optional) geometric mask to world layers/actors (not UI)
        this.layerFloor.setMask(mask);

        // --- Build occluder rectangles from wall tiles ---
        const rects = getBigRectsFromWallLayer(this.layerWalls);

        if (DEBUG) {
            const rectGraphics = this.add.graphics({ fillStyle: { color: 0x0000aa } }).setDepth(DEPTH_UI - 1);
            for (const r of rects) rectGraphics.fillRectShape(r);
            const rectVertGraphics = this.add.graphics({ fillStyle: { color: 0x00aaaa } }).setDepth(DEPTH_UI - 1);
            for (const r of rects) for (const v of getRectVertices(r)) rectVertGraphics.fillPointShape(v, 4);
            console.log('rect count', rects.length);
        }

        // Convert rectangles to edges/vertices for raycast
        this.edges = rects.flatMap(getRectEdges);
        this.vertices = rects.flatMap(getRectVertices);
        this.rays = this.vertices.map(() => new Line());

        // UI
        this.addMobileButtons();
    }

    update (time, delta) {
        // Movement
        const move = 300;
        const joy = this.joystick?.createCursorKeys?.() || {left:{isDown:false},right:{isDown:false},up:{isDown:false},down:{isDown:false}};

        this.player.body.setVelocity(0);
        if (this.cursors.left.isDown || joy.left.isDown)  this.player.body.setVelocityX(-move);
        else if (this.cursors.right.isDown || joy.right.isDown) this.player.body.setVelocityX(move);

        if (this.cursors.up.isDown || joy.up.isDown)      this.player.body.setVelocityY(-move);
        else if (this.cursors.down.isDown || joy.down.isDown) this.player.body.setVelocityY(move);

        if (this.cursors.left.isDown || joy.left.isDown)
        {
            this.player.setAngle(0).setFlipX(true);
            this.player.anims.play('left', true);
            this.direction='left';
            this.isMoving = true;
        }
        else if (this.cursors.right.isDown || joy.right.isDown)
        {
            this.player.anims.play('right',true);
            this.player.setAngle(0).setFlipX(false);
            this.direction='right';
            this.isMoving = true;
        }
        else if (this.cursors.up.isDown || joy.up.isDown)
        {
            this.player.anims.play('up',   true);
            this.player.setAngle(-90).setFlipX(false);
            this.direction='up';
            this.isMoving = true;
        }
        else if (this.cursors.down.isDown || joy.down.isDown)
        {
            this.player.anims.play('down', true);
            this.player.setAngle(90).setFlipX(false);
            this.direction='down';
            this.isMoving = true;
        }
        else
        {
            this.player.anims.stop();
            this.isMoving = false;
        }

        // Lighting (spotlights + bullet glow)
        this.updateMaskLight();

        // Raycast dynamic shadows
        this.updateMaskRaycast();

        if (this.lastMoveSentTime + this.moveCommandInterval < time) {
            this.sendGameCommand('PlayerMoveCommand', {
                x: this.player.x,
                y: this.player.y,
                direction: this.direction,
                isMoving: this.isMoving
            });
            this.lastMoveSentTime = time;
        }

        for (const id in this.players) {
            const p = this.players[id];
            if (p.hpText) {
                p.hpText.x = p.x;
                p.hpText.y = p.y + 20;
            }
        }

        if (this.player.hpText) {
            this.player.hpText.x = this.player.x;
            this.player.hpText.y = this.player.y + 20;
        }
    }

    onIncomingGameEvent (name, data) {
        if (name === 'CreaturesStatsUpdateEvent') {
            for (const p of data.players) {
                const id = p.clientId;
                if (id === this.myClientId) {
                    if (this.player.hp !== p.hp) {
                        this.player.hp = p.hp;
                        if (this.player.hpText) {
                            this.player.hpText.setText(p.hp + '/100');
                        }
                    }
                    continue;
                }
                let justSpawned = false;
                if (!this.players[id]) {
                    // spawn new player
                    const np = this.physics.add.sprite(p.x, p.y, 'player', 1).setScale(3.5).setDepth(DEPTH_PLAYER);
                    np.id = id;
                    np.hp = p.hp;

                    np.setTint(Math.random() * 0xffffff);

                    np.hpText = this.add.text(p.x, p.y, p.hp + '/100', { font: '8px Arial', fill: '#ffffff' }).setOrigin(0.5, 1).setDepth(DEPTH_PLAYER + 1);

                    this.players[id] = np;
                    this.bullets.addPlayer(np);

                    justSpawned = true;
                    console.log('spawn player', id, Object.keys(this.players).length);
                }

                this.updatePlayerPos(p);

                const pSprite = this.players[id];
                if (pSprite.hp !== p.hp || justSpawned) {
                    pSprite.hp = p.hp;
                    if (pSprite.hpText) {
                        pSprite.hpText.setText(p.hp + '/100');
                    }
                    if (p.hp === 0) {
                        if (pSprite.hpText) {
                            pSprite.hpText.destroy();
                            pSprite.setTint(0xff3333);
                            pSprite.setDepth(DEPTH_DEAD);
                            pSprite.disableBody();
                        }
                    }
                }
            }

            for (const m of data.monsters) {
                const id = m.id;
                let justSpawned = false;
                if (!this.monsters[id]) {
                    // spawn new monster
                    const nm = this.physics.add.sprite(m.x, m.y, m.kind, 0).setScale(2).setDepth(DEPTH_MONSTER);
                    nm.id = id;
                    nm.hp = m.hp;
                    nm.hpText = this.add.text(m.x, m.y, m.hp + '/100', { font: '14px Arial', fill: '#ffffff' }).setOrigin(0.5, 1).setDepth(DEPTH_MONSTER + 1);
                    this.monsters[id] = nm;
                    this.bullets.addMonster(nm);
                    justSpawned = true;
                    console.log('spawn monster', id, Object.keys(this.monsters).length);
                }

                this.updateMonsterPos(m);

                const mSprite = this.monsters[id];

                if (mSprite.hp !== m.hp || justSpawned) {
                    mSprite.hp = m.hp;
                    if (mSprite.hpText) {
                        mSprite.hpText.setText(m.hp + '/100');
                    }
                    if (m.hp === 0) {
                        if (mSprite.hpText) {
                            mSprite.hpText.destroy();
                            mSprite.setTint(0x333333);
                            mSprite.setDepth(DEPTH_DEAD);
                            mSprite.disableBody();
                        }
                    }
                }
            }

            return;
        }
        if (name === 'CreaturesPosUpdateEvent') {
            for (const p of data.players) {
                this.updatePlayerPos(p)
            }

            for (const m of data.monsters) {
                this.updateMonsterPos(m);
            }

            return;
        }

        if (name === 'FireballEvent') {
            this.bullets.fireBullet(data.clientId, data.x, data.y, data.direction)
        }

        if (name === 'PlayerDeathEvent') {
            if (data.clientId === this.myClientId) {
                this.add.text(this.scale.width / 2, this.scale.height / 2, 'YOU DIED', { font: '24px Arial', fill: '#ff0000' })
                    .setOrigin(0.5, 0.5)
                    .setScrollFactor(0, 0)
                    .setDepth(DEPTH_UI);
                this.isDead = true;
                console.log('this is my death');
            }
        }
        if (name === 'ArrowEvent') {
            const ar = this.physics.add.sprite(data.x1, data.y1, 'archer', 0).setScale(0.4);
            const dir = new Phaser.Math.Vector2(data.x2 - data.x1, data.y2 - data.y1).normalize();
            ar.setVelocity(dir.x * 400, dir.y * 400);
            this.physics.add.collider(ar, this.layerWalls, () => ar.destroy(), null, this);
            this.physics.add.overlap(ar, this.player, () => {
                ar.destroy();
                this.sendGameCommand('HitPlayerCommand', {
                    monsterId: data.monsterId,
                    targetClientId: this.myClientId
                });
            }, null, this);
        }

        console.log('INCOMING GAME EVENT', name, data);
    }

    updatePlayerPos(p)
    {
        if (p.clientId === this.myClientId) {
            return;
        }

        const pSprite = this.players[p.clientId];
        if (!pSprite) {
            return;
        }

        pSprite.x = p.x;
        pSprite.y = p.y;
        const direction = p.direction;
        switch (direction) {
            case 'up': pSprite.anims.play('up', true); break;
            case 'down': pSprite.anims.play('down', true); break;
            case 'left': pSprite.anims.play('left', true); break;
            case 'right': pSprite.anims.play('right', true); break;
        }

        if (!p.isMoving) {
            pSprite.anims.stop();
        }
    }

    updateMonsterPos(m)
    {
        const mSprite = this.monsters[m.id];
        if (!mSprite) {
            return;
        }

        mSprite.x = m.x;
        mSprite.y = m.y;
    }

    castFireball() {
        this.sendGameCommand('CastFireballCommand', {
            x: this.player.x,
            y: this.player.y,
            direction: this.direction,
        });
    }

    onBulletHitPlayer(bullet, player)
    {
        console.log('hit player', bullet.clientId, player.id);
        if (bullet.clientId !== this.myClientId) {
            return; // only report hits caused by our own bullets
        }
        this.sendGameCommand('HitPlayerCommand', {
            originClientId: bullet.clientId,
            targetClientId: player.id
        });
    }

    onBulletHitMonster(bullet, monster)
    {
        console.log('hit monster', bullet.clientId, monster.id, this.myClientId);
        if (bullet.clientId !== this.myClientId) {
            return; // only report hits caused by our own bullets
        }
        this.sendGameCommand('HitMonsterCommand', {
            originClientId: bullet.clientId,
            monsterId: monster.id
        });
        console.log('monster hit command sent');
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

        // Sample bullets periodically to build a short-lived glow trail
        const now = this.time.now;
        if (now - this._lastBulletSampleAt >= BULLET_SAMPLE_MS) {
            this._lastBulletSampleAt = now;
            for (const s of this._iterateAliveBullets()) {
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
        draw(this.graphics, calc(this.player, this.vertices, this.edges, this.rays), this.rays, this.edges);
    }

    addMobileButtons () {
        if (this.joystick) this.joystick.destroy(true, true);

        const joyStickConfig = {
            x: 85,
            y: 600 * this.uiScaleY - 85,
            radius: 100,
            base: this.add.circle(0, 0, 80, 0x888888, 0.3).setDepth(DEPTH_UI),
            thumb: this.add.circle(0, 0, 40, 0xcccccc, 0.3).setDepth(DEPTH_UI),
            dir: '8dir'
        };

        this.joystick = this.plugins.get('rexvirtualjoystickplugin').add(this, joyStickConfig);
        if (this.joystick.base?.setDepth)  this.joystick.base.setDepth(DEPTH_UI);
        if (this.joystick.thumb?.setDepth) this.joystick.thumb.setDepth(DEPTH_UI);

        const btnScale = Math.max(this.uiScaleX, this.uiScaleY);

        const buttonFire = this.add.sprite(this.scale.width - 85 * this.uiScaleX, this.scale.height - 85 * this.uiScaleY, 'controls', 'fire2');
        buttonFire.setAlpha(0.3).setScrollFactor(0, 0).setScale(btnScale).setInteractive({ useHandCursor: true }).setDepth(DEPTH_UI);
        buttonFire.on('pointerdown', () => this.castFireball());

        if (this.sys.game.device.fullscreen.available) {
            const buttonFs = this.add.sprite(this.scale.width - 85 * this.uiScaleX, 40 * this.uiScaleY, 'controls', 'fullscreen1');
            buttonFs.setAlpha(0.3).setScrollFactor(0, 0).setInteractive({ useHandCursor: true }).setDepth(DEPTH_UI);
            buttonFs.on('pointerup', () => {
                if (this.scale.isFullscreen) this.scale.stopFullscreen();
                else this.scale.startFullscreen();
            });
        }
    }

    // --- bullets helpers ---
    _iterateAliveBullets() {
        const out = [];
        const b = this.bullets;
        if (!b) return out;
        const arr = b.group?.getChildren?.() ?? b.children?.entries ?? b.children ?? b.getChildren?.() ?? [];
        for (const s of arr) if (s?.active) out.push(s);
        return out;
    }

    _pushBulletLight(x, y, expiresAt) {
        this.bulletGlowTrail.push({ x, y, expiresAt });
        if (this.bulletGlowTrail.length > BULLET_TRAIL_CAP) {
            this.bulletGlowTrail.splice(0, this.bulletGlowTrail.length - BULLET_TRAIL_CAP);
        }
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

function calc (source, vertices, edges, rays) {
    const sx = source.x, sy = source.y;
    return sortClockwise(
        rays.map((ray, i) => {
            ray.setTo(sx, sy, vertices[i].x, vertices[i].y);
            Extend(ray, 0, 1000);
            for (const edge of edges) getRayToEdge(ray, edge);
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

// ===================== Utils: Walls to rects =====================
function getBigRectsFromWallLayer(layer) {
    const mapW = layer.layer.width;
    const mapH = layer.layer.height;
    const tw = layer.tilemap.tileWidth;
    const th = layer.tilemap.tileHeight;

    const isSolidAt = (x, y) => {
        const t = layer.getTileAt(x, y);
        return !!t && (t.collides === true || t.properties?.collides === true);
    };

    // 1) horizontal runs per row
    const runs = Array.from({ length: mapH }, () => []);
    for (let y = 0; y < mapH; y++) {
        let x = 0;
        while (x < mapW) {
            if (!isSolidAt(x, y)) { x++; continue; }
            const x0 = x;
            while (x < mapW && isSolidAt(x, y)) x++;
            runs[y].push({ x: x0, w: x - x0 });
        }
    }

    // 2) vertical merge of identical runs
    const rects = [];
    const used = runs.map(row => row.map(() => false));

    for (let y = 0; y < mapH; y++) {
        for (let i = 0; i < runs[y].length; i++) {
            if (used[y][i]) continue;
            const { x: rx, w: rw } = runs[y][i];
            let h = 1;
            // try to extend downwards while the exact same run exists and not used
            let yy = y + 1;
            while (yy < mapH) {
                let foundIdx = -1;
                for (let j = 0; j < runs[yy].length; j++) {
                    if (!used[yy][j] && runs[yy][j].x === rx && runs[yy][j].w === rw) { foundIdx = j; break; }
                }
                if (foundIdx === -1) break;
                used[yy][foundIdx] = true;
                h++;
                yy++;
            }
            used[y][i] = true;

            const t0 = layer.getTileAt(rx, y);
            rects.push(new Rectangle(t0.getLeft(), t0.getTop(), rw * tw, h * th));
        }
    }

    return rects;
}