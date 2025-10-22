const DEBUG = false;

// Layering (z-index)
const DEPTH_DEAD = 10;
const DEPTH_OBJECTS = 20;
const DEPTH_PLAYER = 30;
const DEPTH_MONSTER = 40;
const DEPTH_PROJECTILES = 200;
const DEPTH_UPPER_WALLS = 500;
const DEPTH_DARKNESS = 9000;
const DEPTH_UI = 10000;

const PLAYER_SCALE = 1.5;

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
    projectiles;

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
    isMoving = false;
    isDead = false;

    lastMoveSentTime = 0;
    moveCommandInterval = 1000 / 60; // ms

    players = {};
    monsters = {};
    gameObjects = {};

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

        this.layerFloor = this.map.createLayer('floor', tiles, 0, 0);
        this.layerWalls = this.map.createLayer('walls', tiles, 0, 0);
        this.layerWalls.setCollisionByProperty({ collides: true });

        const layerWallsUpper = this.map.createBlankLayer('upper_walls', tiles, 0, 0, this.map.width, this.map.height);
        this.layerWalls.forEachTile((tile) => {
            if (tile.properties.extra_z === true) {
                layerWallsUpper.putTileAt(tile, tile.x, tile.y);
                this.layerWalls.removeTileAt(tile.x, tile.y);
            }
        });
        layerWallsUpper.setDepth(DEPTH_UPPER_WALLS);

        this.player = new MyPlayer('mage', this, gameData.playerData);
        this.physics.add.collider(this.player, this.layerWalls);

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
        if (DEBUG) {
            this.graphics.setAlpha(0.5);
            this.add.existing(this.graphics);
        } else {
            this.mask = new Phaser.Display.Masks.GeometryMask(this, this.graphics);
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
        this.input.addPointer(1); // allow 2 simultaneous pointers for mobile
        this.addMobileButtons();
    }

    update (time, delta) {
        // Movement
        const move = 200;
        const joy = this.joystick?.createCursorKeys?.() || {left:{isDown:false},right:{isDown:false},up:{isDown:false},down:{isDown:false}};

        this.player.body.setVelocity(0);
        if (this.cursors.left.isDown || joy.left.isDown)  this.player.body.setVelocityX(-move);
        else if (this.cursors.right.isDown || joy.right.isDown) this.player.body.setVelocityX(move);

        if (this.cursors.up.isDown || joy.up.isDown)      this.player.body.setVelocityY(-move);
        else if (this.cursors.down.isDown || joy.down.isDown) this.player.body.setVelocityY(move);

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

        if (this.isMoving) {
            this.player.playMoveAnimation(this.direction);
        } else {
            this.player.playIdleAnimation(this.direction);
        }

        // Lighting (spotlights + bullet glow)
        this.updateMaskLight();

        // Raycast dynamic shadows
        this.updateMaskRaycast();

        if (this.lastMoveSentTime + this.moveCommandInterval < time) {
            this.sendGameCommand('PlayerMoveCommand', {
                x: Math.round(this.player.x),
                y: Math.round(this.player.y),
                direction: this.direction,
                isMoving: this.isMoving
            });
            this.lastMoveSentTime = time;
        }
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
        if (bullet.monsterId && player.id === this.myClientId) {
            // hit caused by monster's bullet on ourselves
            this.sendGameCommand('HitPlayerCommand', {
                monsterId: bullet.monsterId,
                targetClientId: this.myClientId
            });
            return;
        }

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
        console.log('hit monster', bullet.clientId, this.myClientId, bullet.monsterId, monster.id);
        if (bullet.monsterId) {
            return; // ignore hitting monsters by monster bullets
        }

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

                draw(this.graphics, calc(this.player, vertices, edges, rays), rays, edges);

                return;
            }
        }
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
        for (const area of rectsByAreas) {
            aid = area.id;
            if (r.properties["area_" + aid]) {
                area.rects.push(new Rectangle(r.x, r.y, r.width, r.height));
            }
        }
    }

    return rectsByAreas;
}
