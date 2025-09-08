// Disable mask and draw geometry
const DEBUG = true;

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

    constructor ()
    {
        super({ key: 'Game' });
    }

    create ()
    {
        this.map = this.make.tilemap({ key: 'map' });

        const tiles = this.map.addTilesetImage('tiles_atlas', 'tiles');

        this.layerFloor = this.map.createLayer(0, tiles, 0, 0); // floor
        this.layerWalls = this.map.createLayer(1, tiles, 0, 0); // walls
        // all tiles can collide, we just use collider for layer
        this.map.setCollisionBetween(0, 5);

        const mapRects = this.map.getObjectLayer('rects')['objects'];

        this.player = this.physics.add.sprite(120, 140, 'player', 1);
        this.player.setScale(4);

        this.physics.add.collider(this.player, this.layerWalls);

        this.cameras.main.setBounds(0, 0, this.map.widthInPixels, this.map.heightInPixels);
        this.cameras.main.startFollow(this.player);

        this.cursors = this.input.keyboard.createCursorKeys();

        // https://phaser.io/examples/v3.85.0/tilemap/collision/view/tilemap-spotlight
        //this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
        // TODO: fix scale
        this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
        // this.scale.on('resize', (gameSize, baseSize, displaySize, resolution) => {
        //
        //     this.rt.setSize(1200, 900);
        // })

        //  Make sure it doesn't scroll with the camera
        this.rt.setOrigin(0, 0);
        this.rt.setScrollFactor(0, 0);

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
        const wallsRects = [];
        for (let i = 0; i < mapRects.length; i++) {
            const rect = mapRects[i];
            wallsRects.push(new Rectangle(rect.x, rect.y, rect.width, rect.height));
        }

        // Rectangles, will form the edges
        const rects = wallsRects;

        // Convert rectangles into edges (line segments)
        this.edges = rects.flatMap(getRectEdges);

        // Convert rectangles into vertices
        this.vertices = rects.flatMap(getRectVertices);

        // One ray will be sent through each vertex
        this.rays = this.vertices.map(() => new Line());

        // Draw the mask once
        //draw(this.graphics, calc(this.player, this.vertices, this.edges, this.rays), this.rays, this.edges);

        this.scaleX = this.scale.width / 800;
        this.scaleY = this.scale.height / 600;
        console.log('scales', this.scaleX, this.scaleY);

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
        //this.updateMaskRaycast();
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
        const posLeftX = 100;
        const posBottomY = 600 * this.scaleY - 100;

        const container = this.add.container();
        container.setAlpha(0.6);
        container.setScrollFactor(0, 0);

        const buttonLeft = this.add.sprite(posLeftX, posBottomY, 'controls', 'left1');
        container.add(buttonLeft);
        buttonLeft.setOrigin(1, 0.5);
        buttonLeft.setInteractive({ useHandCursor: true });
        buttonLeft.on('pointerdown', () => this.moveLeft = true);
        buttonLeft.on('pointerup', () => this.moveLeft = false);

        const buttonRight = this.add.sprite(posLeftX, posBottomY, 'controls', 'right1');
        container.add(buttonRight);
        buttonRight.setOrigin(0, 0.5);
        buttonRight.setInteractive({ useHandCursor: true });
        buttonRight.on('pointerdown', () => this.moveRight = true);
        buttonRight.on('pointerup', () => this.moveRight = false);

        const buttonDown = this.add.sprite(posLeftX, posBottomY, 'controls', 'down1');
        container.add(buttonDown);
        buttonDown.setOrigin(0.5, 0);
        buttonDown.setInteractive({ useHandCursor: true });
        buttonDown.on('pointerdown', () => this.moveDown = true);
        buttonDown.on('pointerup', () => this.moveDown = false);

        const buttonUp = this.add.sprite(posLeftX, posBottomY, 'controls', 'up1');
        container.add(buttonUp);
        buttonUp.setOrigin(0.5, 1);
        buttonUp.setInteractive({ useHandCursor: true });
        buttonUp.on('pointerdown', () => this.moveUp = true);
        buttonUp.on('pointerup', () => this.moveUp = false);

        if (this.sys.game.device.fullscreen.available) {
            const buttonFs = this.add.sprite(800 * this.scaleX - 30, 30, 'controls', 'fullscreen1');
            container.add(buttonFs);
            buttonFs.setOrigin(1, 0);

            buttonFs.setInteractive({ useHandCursor: true });

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
    graphics
        .clear()
        .fillStyle(FILL_COLOR)
        .fillPoints(vertices, true);

    if (DEBUG) {
        // for (const ray of rays) {
        //     graphics.strokeLineShape(ray);
        // };
        // for (const edge of edges) {
        //     graphics.strokeLineShape(edge);
        // };

        graphics.fillStyle(DEBUG_FILL_COLOR);

        for (const vert of vertices) {
            graphics.fillPointShape(vert, 4);
        };
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
