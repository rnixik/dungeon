// Disable mask and draw geometry
const DEBUG = false;

// Colors
const BLACK = 0;
const WHITE = 0xffffff;
const FILL_COLOR = BLACK;
const DEBUG_STROKE_COLOR = WHITE;
const DEBUG_FILL_COLOR = 0xff0000;

// Mask
const LIGHT_RADIUS   = 320;
const LIGHT_FEATHER  = 128;    // thikness of shade
const LIGHT_ALPHA    = 0.85;      // density of darkness
const DARKNESS_DEPTH = 9000;  // darkness below
const UI_DEPTH       = 10000; // UI above everything

// Shortcuts
const { Circle, Line, Point, Rectangle } = Phaser.Geom;
const { EPSILON } = Phaser.Math;
const { Extend } = Line;
const { ContainsPoint } = Rectangle;
const { LineToLine } = Phaser.Geom.Intersects;

class Game extends Phaser.Scene
{
    player;
    cursors;
    rt;
    map;
    layerWalls;
    layerFloor;
    vertices;
    edges;
    rays;
    graphics;
    direction = 'right';
    scaleX;
    scaleY;
    joystick;
    bullets;
    otherPlayer;
    otherPlayer2;

    constructor ()
    {
        super({ key: 'Game' });
    }

    create ()
    {
        this.scaleX = this.scale.width / 800;
        this.scaleY = this.scale.height / 600;
        console.log('scales', this.scaleX, this.scaleY);

        //#region Setup environment
        this.map = this.make.tilemap({ key: 'map' });
        const tiles = this.map.addTilesetImage('environment', 'tiles');

        this.layerFloor = this.map.createLayer('floor', tiles, 0, 0);
        this.layerWalls = this.map.createLayer('walls', tiles, 0, 0);

        this.layerWalls.setCollisionByProperty({ collides: true });
        //#endregion

        //#region Setup player
        this.player = this.physics.add.sprite(120, 140, 'player', 1);
        this.player.setScale(3.5);

        this.physics.add.collider(this.player, this.layerWalls);
        this.bullets = new Bullets(this, this.layerWalls);

        this.otherPlayer = this.add.sprite(1000, 100, 'player', 1);
        this.otherPlayer.setScale(3.5);

        this.otherPlayer2 = this.add.sprite(400, 400, 'player', 1);
        this.otherPlayer2.setScale(3.5);
        //#endregion


        this.cameras.main.setBounds(0, 0, this.map.widthInPixels, this.map.heightInPixels);
        this.cameras.main.startFollow(this.player);



        this.cursors = this.input.keyboard.createCursorKeys();

        // https://phaser.io/examples/v3.85.0/tilemap/collision/view/tilemap-spotlight
        this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
        this.rt.setOrigin(0, 0);
        this.rt.setScrollFactor(0, 0);

        this.rt.setDepth(this.DARKNESS_DEPTH);
        this.rt.setAlpha(LIGHT_ALPHA);
        this.rt.setDepth(1000);

        this.lightRadius = LIGHT_RADIUS;

        const d = this.lightRadius;
        const canvas = this.textures.createCanvas('lightCanvas', d, d).getContext();
        const gradient = canvas.createRadialGradient(d/2, d/2, d/2 - LIGHT_FEATHER, d/2, d/2, d/2);

        gradient.addColorStop(0, 'rgba(255,255,255,1)');
        gradient.addColorStop(1, 'rgba(255,255,255,0)'); //funny if comment this line

        canvas.fillStyle = gradient;
        canvas.fillRect(0, 0, d, d);

        this.textures.get('lightCanvas').refresh();
        this.maskKey = 'lightCanvas';

        this.scale.on('resize', (gameSize, baseSize, displaySize, resolution) => {
            console.log('new size', this.scale.width, this.scale.height);
            //this.rt.setSize(this.scale.width, this.scale.height);

            // setting new size doesn't work properly, so we destroy and create new
            this.rt.destroy();
            this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
            this.rt.setOrigin(0, 0);
            this.rt.setScrollFactor(0, 0);
            this.rt.setDepth(DARKNESS_DEPTH);

            // buttons need to be repositioned because they will be hidden by light mask
            this.addMobileButtons();
        })

        this.graphics = this.make.graphics({ lineStyle: { color: DEBUG_STROKE_COLOR, width: 0.5 } });

        let mask;

        if (DEBUG) {
            mask = null;
            this.graphics.setAlpha(0.5);
            this.add.existing(this.graphics);
        } else {
            mask = new Phaser.Display.Masks.GeometryMask(this, this.graphics);
        }

        // Mask objects and background.
        this.layerFloor.setMask(mask);
        this.otherPlayer.setMask(mask);
        this.otherPlayer2.setMask(mask);

        // Create Rectangles from wall tiles
        const rects = getBigRectsFromWallLayer(this.layerWalls);

        // fill debug rects
        if (DEBUG) {
            const rectGraphics = this.add.graphics({ fillStyle: { color: 0x0000aa } });
            for (const rect of rects) {
                rectGraphics.fillRectShape(rect);
            }

            const rectVertGraphics = this.add.graphics({ fillStyle: { color: 0x00aaaa } });
            for (const rect of rects) {
                const verts = getRectVertices(rect);
                for (const vert of verts) {
                    rectVertGraphics.fillPointShape(vert, 4);
                }
            }

            console.log('rect length', rects.length);
        }

        // Convert rectangles into edges and vertices (line segments)
        this.edges = rects.flatMap(getRectEdges);
        this.vertices = rects.flatMap(getRectVertices);

        // One ray will be sent through each vertex
        this.rays = this.vertices.map(() => new Line());

        // Draw the mask once
        //draw(this.graphics, calc(this.player, this.vertices, this.edges, this.rays), this.rays, this.edges);

        this.addMobileButtons();

        const spaceBar = this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.SPACE);
        spaceBar.on('down', () => this.bullets.fireBullet(this.player.x, this.player.y, this.direction));
    }

    update (time, delta)
    {
        this.player.body.setVelocity(0);
        const moveSpeed = 300;
        const joystick = this.joystick.createCursorKeys();

        // Horizontal movement
        if (this.cursors.left.isDown || joystick.left.isDown)
        {
            this.player.body.setVelocityX(-1 * moveSpeed);
        }
        else if (this.cursors.right.isDown || joystick.right.isDown)
        {
            this.player.body.setVelocityX(moveSpeed);
        }

        // Vertical movement
        if (this.cursors.up.isDown || joystick.up.isDown)
        {
            this.player.body.setVelocityY(-1 * moveSpeed);
        }
        else if (this.cursors.down.isDown || joystick.down.isDown)
        {
            this.player.body.setVelocityY(moveSpeed);
        }

        // Update the animation last and give left/right animations precedence over up/down animations
        if (this.cursors.left.isDown || joystick.left.isDown)
        {
            this.player.anims.play('left', true);
            this.direction = 'left';
        }
        else if (this.cursors.right.isDown || joystick.right.isDown)
        {
            this.player.anims.play('right', true);
            this.direction = 'right';
        }
        else if (this.cursors.up.isDown || joystick.up.isDown)
        {
            this.player.anims.play('up', true);
            this.direction = 'up';
        }
        else if (this.cursors.down.isDown || joystick.down.isDown)
        {
            this.player.anims.play('down', true);
            this.direction = 'down';
        }
        else
        {
            this.player.anims.stop();
        }

        // it is light aroung the player but works through walls
        this.updateMaskLight();

        // if we want to hide distant walls, we can alt alpha based on distance from player
        // this.updateAlphaOnMap();

        // it makes dynamic shadows
        this.updateMaskRaycast();
    }

    updateMaskLight() {
        const cam  = this.cameras.main;
        const half = this.lightRadius / 2;

        this.rt.clear();
        this.rt.fill(0x000000, 1);

        this.rt.erase(this.maskKey, (this.player.x - half) - cam.scrollX, (this.player.y - half) - cam.scrollY);
        this.rt.erase(this.maskKey, (this.otherPlayer.x - half) - cam.scrollX, (this.otherPlayer.y - half) - cam.scrollY);
        this.rt.erase(this.maskKey, (this.otherPlayer2.x - half) - cam.scrollX, (this.otherPlayer2.y - half) - cam.scrollY);
    }

    updateAlphaOnMap ()
    {
        const cam = this.cameras.main;
        const origin = this.layerFloor.getTileAtWorldXY(this.player.x, this.player.y, false, cam);

        this.layerWalls.forEachTile(tile =>
        {
            const dist = Phaser.Math.Distance.Chebyshev(
                origin.x,
                origin.y,
                tile.x,
                tile.y
            );

            tile.setAlpha(1 - 0.2 * dist);
        });
    }

    updateMaskRaycast ()
    {
        draw(this.graphics, calc(this.player, this.vertices, this.edges, this.rays), this.rays, this.edges);
    }

    addMobileButtons ()
    {
        if (this.joystick) {
            this.joystick.destroy(true, true);
        }

        const joyStickConfig = {
            x: 85,
            y: 600 * this.scaleY - 85,
            radius: 100,
            base: this.add.circle(0, 0, 80, 0x888888, 0.3).setDepth(UI_DEPTH),
            thumb: this.add.circle(0, 0, 40, 0xcccccc, 0.3).setDepth(UI_DEPTH),
            dir: '8dir'
        };

        this.joystick = this.plugins.get('rexvirtualjoystickplugin').add(this, joyStickConfig);

        const buttonFire = this.add.sprite(this.scale.width - 85 * this.scaleX, this.scale.height - 85 * this.scaleY, 'controls', 'fire2');
        buttonFire.setAlpha(0.3);
        buttonFire.setScrollFactor(0, 0);
        buttonFire.setScale(Math.max(this.scaleX, this.scaleY));
        buttonFire.setInteractive({ useHandCursor: true });
        buttonFire.on('pointerdown', () => this.bullets.fireBullet(this.player.x, this.player.y, this.direction));
        buttonFire.setDepth(UI_DEPTH);

        if (this.sys.game.device.fullscreen.available) {
            const buttonFs = this.add.sprite(this.scale.width - 85 * this.scaleX, 40 * this.scaleY, 'controls', 'fullscreen1');
            buttonFs.setAlpha(0.3);
            buttonFs.setScrollFactor(0, 0);
            buttonFs.setInteractive({ useHandCursor: true });
            buttonFs.setDepth(UI_DEPTH);

            buttonFs.on('pointerup', function (){
                if (this.scale.isFullscreen) {
                    this.scale.stopFullscreen();
                } else {
                    this.scale.startFullscreen();
                }
            }, this);
        }
    }
}

var sceneConfigGame = new Game();

function getTilesBigRects(tileLayer) {
    const rects = [];

    tileLayer.forEachTile((tile) => {
        if (tile.index === -1) return;

        const worldX = tile.getLeft();
        const worldY = tile.getTop();
        const width = tile.width;
        const height = tile.height;

        rects.push(new Rectangle(worldX, worldY, width, height));
    });

    return rects;
}

// Draw the mask shape, from vertices
function draw (graphics, vertices, rays, edges) {
    if (vertices.length < 3) {
        graphics.clear()
        return;
    }

    graphics
        .clear()
        .fillStyle(FILL_COLOR)
        .fillPoints(vertices, true);

    if (DEBUG) {
        for (const ray of rays) {
            graphics.strokeLineShape(ray);
        }
        for (const edge of edges) {
            graphics.strokeLineShape(edge);
        }

        graphics.fillStyle(DEBUG_FILL_COLOR);

        for (const vert of vertices) {
            graphics.fillPointShape(vert, 4);
        }
    }
}

// Place the rays, calculate and return intersections.
function calc (source, vertices, edges, rays) {
    const sx = source.x;
    const sy = source.y;

    // Sort clockwise …
    return sortClockwise(
        // each ray-edge intersection, or the ray's endpoint if no intersection
        rays.map((ray, i) => {
            // placing the ray between the source and one vertex …
            ray.setTo(sx, sy, vertices[i].x, vertices[i].y);

            // extended through the wall vertex
            Extend(ray, 0, 1000);

            // placing its endpoint at the intersection with an edge, if any
            for (const edge of edges) {
                getRayToEdge(ray, edge);
            }

            // the new endpoint
            return ray.getPointB();
        }),
        source
    );
}

function getSpriteRect (sprite) {
    const {displayWidth, displayHeight} = sprite;

    return new Rectangle(
        sprite.x - sprite.originX * displayWidth,
        sprite.y - sprite.originY * displayHeight,
        displayWidth,
        displayHeight
    );
}

function getRectEdges (rect) {
    return [
        rect.getLineA(),
        rect.getLineB(),
        rect.getLineC(),
        rect.getLineD()
    ];
}

function getRectVertices (rect) {
    const { left, top, right, bottom } = rect;

    const left1 = left + EPSILON;
    const top1 = top + EPSILON;
    const right1 = right - EPSILON;
    const bottom1 = bottom - EPSILON;
    const left2 = left - EPSILON;
    const top2 = top - EPSILON;
    const right2 = right + EPSILON;
    const bottom2 = bottom + EPSILON;

    return [
        new Point(left1, top1),
        new Point(right1, top1),
        new Point(right1, bottom1),
        new Point(left1, bottom1),
        new Point(left2, top2),
        new Point(right2, top2),
        new Point(right2, bottom2),
        new Point(left2, bottom2)
    ];
}

// If a ray intersects with an edge, place the ray endpoint there and return the intersection.
function getRayToEdge (ray, edge, out) {
    if (!out) out = new Point();

    if (LineToLine(ray, edge, out)) {
        ray.x2 = out.x;
        ray.y2 = out.y;

        return out;
    }

    return null;
}

function sortClockwise (points, center) {
    // Adapted from <https://stackoverflow.com/a/6989383/822138> (ciamej)

    var cx = center.x;
    var cy = center.y;

    var sort = function (a, b) {
        if (a.x - cx >= 0 && b.x - cx < 0) {
            return -1;
        }

        if (a.x - cx < 0 && b.x - cx >= 0) {
            return 1;
        }

        if (a.x - cx === 0 && b.x - cx === 0) {
            if (a.y - cy >= 0 || b.y - cy >= 0) {
                return (a.y > b.y) ? 1 : -1;
            }

            return (b.y > a.y) ? 1 : -1;
        }

        // Compute the cross product of vectors (center -> a) * (center -> b)
        var det = (a.x - cx) * -(b.y - cy) - (b.x - cx) * -(a.y - cy);

        if (det < 0) {
            return -1;
        }

        if (det > 0) {
            return 1;
        }

        // Points a and b are on the same line from the center
        // Check which point is closer to the center
        var d1 = (a.x - cx) * (a.x - cx) + (a.y - cy) * (a.y - cy);
        var d2 = (b.x - cx) * (b.x - cx) + (b.y - cy) * (b.y - cy);

        return (d1 > d2) ? -1 : 1;
    };

    return points.sort(sort);
}

// eslint-disable-next-line no-unused-vars
function pointInRectangles (point, rects) {
    return rects.some((rect) => ContainsPoint(rect, point));
}

function getRectsFromTilesInRadius(layer, x, y, radius) {
    const tiles = layer.getTilesWithinWorldXY(x - radius, y - radius, radius * 2, radius * 2);
    const rects = [];

    tiles.forEach((tile) => {
        if (tile.index === -1) return;

        const worldX = tile.getLeft();
        const worldY = tile.getTop();
        const width = tile.width;
        const height = tile.height;

        rects.push(new Rectangle(worldX, worldY, width, height));
    });

    return rects;
}

function getBigRectsFromWallLayer(layer) {
    const rects = [];

    const mapW = layer.layer.width;
    const mapH = layer.layer.height;

    const visited = Array.from({ length: mapH }, () => Array(mapW).fill(false));

    const isSolidAt = (x, y) => {
        const t = layer.getTileAt(x, y);
        // Phaser marks tile.collides = true, plus consider properties.collides
        return !!t && (t.collides === true || t.properties?.collides === true);
    };

    for (let y = 0; y < mapH; y++) {
        for (let x = 0; x < mapW; x++) {
            if (visited[y][x] || !isSolidAt(x, y)) continue;

            // going right
            let w = 1;
            while (x + w < mapW && !visited[y][x + w] && isSolidAt(x + w, y)) w++;

            // going down, while tile is solid
            let h = 1;
            outer: while (y + h < mapH) {
                for (let i = 0; i < w; i++) {
                    if (visited[y + h][x + i] || !isSolidAt(x + i, y + h)) break outer;
                }
                h++;
            }

            // mark as visited
            for (let dy = 0; dy < h; dy++) {
                for (let dx = 0; dx < w; dx++) {
                    visited[y + dy][x + dx] = true;
                }
            }

            // adding rect in coordinates
            const tile = layer.getTileAt(x, y);
            rects.push(new Phaser.Geom.Rectangle(
                tile.getLeft(),
                tile.getTop(),
                w * tile.width,
                h * tile.height
            ));
        }
    }

    return rects;
}
