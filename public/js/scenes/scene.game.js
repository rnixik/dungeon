// Disable mask and draw geometry
const DEBUG = false;

// Colors
const BLACK = 0;
const WHITE = 0xffffff;
const FILL_COLOR = BLACK;
const DEBUG_STROKE_COLOR = WHITE;
const DEBUG_FILL_COLOR = 0xff0000;

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
    moveUp;
    moveDown;
    moveLeft;
    moveRight;
    scaleX;
    scaleY;
    controlsContainer;

    constructor ()
    {
        super({ key: 'Game' });
    }

    create ()
    {
        this.scaleX = this.scale.width / 800;
        this.scaleY = this.scale.height / 600;
        console.log('scales', this.scaleX, this.scaleY);

        this.map = this.make.tilemap({ key: 'map' });

        const tiles = this.map.addTilesetImage('tiles_atlas', 'tiles');

        this.layerFloor = this.map.createLayer(0, tiles, 0, 0); // floor
        this.layerWalls = this.map.createLayer(1, tiles, 0, 0); // walls
        // all tiles can collide, we just use collider for layer
        this.map.setCollisionBetween(0, 5);

        const mapRects = this.map.getObjectLayer('rects')['objects'];

        this.player = this.physics.add.sprite(120, 140, 'player', 1);
        this.player.setScale(3.5);

        this.physics.add.collider(this.player, this.layerWalls);

        this.cameras.main.setBounds(0, 0, this.map.widthInPixels, this.map.heightInPixels);
        this.cameras.main.startFollow(this.player);

        this.cursors = this.input.keyboard.createCursorKeys();

        // https://phaser.io/examples/v3.85.0/tilemap/collision/view/tilemap-spotlight
        this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
        this.rt.setOrigin(0, 0);
        this.rt.setScrollFactor(0, 0);

        this.scale.on('resize', (gameSize, baseSize, displaySize, resolution) => {
            console.log('new size', this.scale.width, this.scale.height);
            //this.rt.setSize(this.scale.width, this.scale.height);

            // setting new size doesn't work properly, so we destroy and create new
            this.rt.destroy();
            this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
            this.rt.setOrigin(0, 0);
            this.rt.setScrollFactor(0, 0);

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
        //this.layerWalls.setMask(mask);
        this.layerFloor.setMask(mask);

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

        // Convert rectangles into edges (line segments)
        this.edges = rects.flatMap(getRectEdges);

        // Convert rectangles into vertices
        this.vertices = rects.flatMap(getRectVertices);

        // One ray will be sent through each vertex
        this.rays = this.vertices.map(() => new Line());

        // Draw the mask once
        //draw(this.graphics, calc(this.player, this.vertices, this.edges, this.rays), this.rays, this.edges);



        this.addMobileButtons();
    }

    update (time, delta)
    {
        this.player.body.setVelocity(0);
        const moveSpeed = 300;

        // Horizontal movement
        if (this.cursors.left.isDown || this.moveLeft)
        {
            this.player.body.setVelocityX(-1 * moveSpeed);
        }
        else if (this.cursors.right.isDown || this.moveRight)
        {
            this.player.body.setVelocityX(moveSpeed);
        }

        // Vertical movement
        if (this.cursors.up.isDown || this.moveUp)
        {
            this.player.body.setVelocityY(-1 * moveSpeed);
        }
        else if (this.cursors.down.isDown || this.moveDown)
        {
            this.player.body.setVelocityY(moveSpeed);
        }

        // Update the animation last and give left/right animations precedence over up/down animations
        if (this.cursors.left.isDown || this.moveLeft)
        {
            this.player.anims.play('left', true);
        }
        else if (this.cursors.right.isDown || this.moveRight)
        {
            this.player.anims.play('right', true);
        }
        else if (this.cursors.up.isDown || this.moveUp)
        {
            this.player.anims.play('up', true);
        }
        else if (this.cursors.down.isDown || this.moveDown)
        {
            this.player.anims.play('down', true);
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

    updateMaskLight ()
    {
        //  Draw the spotlight on the player
        const cam = this.cameras.main;

        //  Clear the RenderTexture
        this.rt.clear();

        //  Fill it in black
        this.rt.fill(0x000000);

        //  Erase the 'mask' texture from it based on the player position
        //  We - 107, because the mask image is 213px wide, so this puts it on the middle of the player
        //  We then minus the scrollX/Y values, because the RenderTexture is pinned to the screen and doesn't scroll
        // Upd: offset is half the mask image width
        this.rt.erase('mask', (this.player.x - 180) - cam.scrollX, (this.player.y - 180) - cam.scrollY);
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
        if (this.controlsContainer) {
            this.controlsContainer.destroy(true, true);
        }

        const posLeftX = 100;
        const posBottomY = 600 * this.scaleY - 100;

        const container = this.add.container();
        container.setAlpha(0.6);
        container.setScrollFactor(0, 0);

        const buttonLeft = this.add.sprite(posLeftX, posBottomY, 'controls', 'left1');
        container.add(buttonLeft);
        buttonLeft.setOrigin(1, 0.5);
        buttonLeft.setScrollFactor(0, 0);
        buttonLeft.setInteractive({ useHandCursor: true });
        buttonLeft.on('pointerdown', () => this.moveLeft = true);
        buttonLeft.on('pointerup', () => this.moveLeft = false);

        const buttonRight = this.add.sprite(posLeftX, posBottomY, 'controls', 'right1');
        container.add(buttonRight);
        buttonRight.setOrigin(0, 0.5);
        buttonRight.setScrollFactor(0, 0);
        buttonRight.setInteractive({ useHandCursor: true });
        buttonRight.on('pointerdown', () => this.moveRight = true);
        buttonRight.on('pointerup', () => this.moveRight = false);

        const buttonDown = this.add.sprite(posLeftX, posBottomY, 'controls', 'down1');
        container.add(buttonDown);
        buttonDown.setOrigin(0.5, 0);
        buttonDown.setScrollFactor(0, 0);
        buttonDown.setInteractive({ useHandCursor: true });
        buttonDown.on('pointerdown', () => this.moveDown = true);
        buttonDown.on('pointerup', () => this.moveDown = false);

        const buttonUp = this.add.sprite(posLeftX, posBottomY, 'controls', 'up1');
        container.add(buttonUp);
        buttonUp.setOrigin(0.5, 1);
        buttonUp.setScrollFactor(0, 0);
        buttonUp.setInteractive({ useHandCursor: true });
        buttonUp.on('pointerdown', () => this.moveUp = true);
        buttonUp.on('pointerup', () => this.moveUp = false);

        if (this.sys.game.device.fullscreen.available) {
            const buttonFs = this.add.sprite(800 * this.scaleX - 30, 30, 'controls', 'fullscreen1');
            container.add(buttonFs);
            buttonFs.setOrigin(1, 0);
            buttonFs.setScrollFactor(0, 0);

            buttonFs.setInteractive({ useHandCursor: true });

            buttonFs.on('pointerup', function (){
                if (this.scale.isFullscreen) {
                    this.scale.stopFullscreen();
                } else {
                    this.scale.startFullscreen();
                }
            }, this);
        }

        this.controlsContainer = container;
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
    const visited = new Set();

    const width = layer.tilemap.width;
    const height = layer.tilemap.height;

    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const tile = layer.getTileAt(x, y);
            if (!tile || tile.index === -1) continue;

            const key = `${x},${y}`;
            if (visited.has(key)) continue;

            // Start a new rectangle
            let rectWidth = 1;
            let rectHeight = 1;

            // Expand to the right
            while (x + rectWidth < width) {
                const nextTile = layer.getTileAt(x + rectWidth, y);
                if (nextTile && nextTile.index !== -1) {
                    visited.add(`${x + rectWidth},${y}`);
                    rectWidth++;
                } else {
                    break;
                }
            }

            // Expand downwards
            let canExpandDown = true;
            while (canExpandDown && (y + rectHeight) < height) {
                for (let i = 0; i < rectWidth; i++) {
                    const nextTile = layer.getTileAt(x + i, y + rectHeight);
                    if (!nextTile || nextTile.index === -1) {
                        canExpandDown = false;
                        break;
                    }
                }
                if (canExpandDown) {
                    for (let i = 0; i < rectWidth; i++) {
                        visited.add(`${x + i},${y + rectHeight}`);
                    }
                    rectHeight++;
                }
            }

            // Mark all tiles in the rectangle as visited
            for (let dy = 0; dy < rectHeight; dy++) {
                for (let dx = 0; dx < rectWidth; dx++) {
                    visited.add(`${x + dx},${y + dy}`);
                }
            }

            // Add the rectangle to the list
            const worldX = tile.getLeft();
            const worldY = tile.getTop();
            const worldWidth = rectWidth * tile.width;
            const worldHeight = rectHeight * tile.height;

            rects.push(new Rectangle(worldX, worldY, worldWidth, worldHeight));
        }
    }

    return rects;
}
