// NavArrows draws HUD compass arrows that point a cultist toward every other
// living player that is currently off-screen. It is a hunting aid for the
// hidden cultist team: fellow cultists are marked purple, everyone else (the
// prey) is drawn in that player's own colour.
//
// Arrows live in screen space (scrollFactor 0) above the fog of war, clamped to
// the inside edge of the viewport and rotated to face their target.
class NavArrows
{
    static TEXTURE_KEY = 'nav_arrow';
    static MARGIN = 46;            // inset from the screen edge, in px
    static ALLY_COLOR = 0xcc33ff;  // fellow cultists

    scene;
    arrows = {};                   // clientId -> Phaser.Image

    constructor(scene)
    {
        this.scene = scene;
        this._ensureTexture();
    }

    _ensureTexture()
    {
        if (this.scene.textures.exists(NavArrows.TEXTURE_KEY)) {
            return;
        }
        const w = 22, h = 18;
        const g = this.scene.make.graphics({ x: 0, y: 0, add: false });
        // Black base acts as an outline; tint multiplies, so black stays black.
        g.fillStyle(0x000000, 1);
        g.fillTriangle(0, 0, 0, h, w, h / 2);
        // White core takes the runtime tint colour.
        g.fillStyle(0xffffff, 1);
        g.fillTriangle(3, 4, 3, h - 4, w - 4, h / 2);
        g.generateTexture(NavArrows.TEXTURE_KEY, w, h);
        g.destroy();
    }

    _getArrow(id)
    {
        if (!this.arrows[id]) {
            this.arrows[id] = this.scene.add.image(0, 0, NavArrows.TEXTURE_KEY)
                .setOrigin(0.5, 0.5)
                .setScrollFactor(0, 0)
                .setDepth(DEPTH_UI + 1)
                .setVisible(false);
        }
        return this.arrows[id];
    }

    update()
    {
        const scene = this.scene;
        const me = scene.player;

        // Only the cultist team gets the radar.
        if (!scene.isCultist || !me) {
            for (const id in this.arrows) {
                this.arrows[id].setVisible(false);
            }
            return;
        }

        const cam = scene.cameras.main;
        const view = cam.worldView;
        const toScreenX = (wx) => (wx - view.x) / view.width * cam.width;
        const toScreenY = (wy) => (wy - view.y) / view.height * cam.height;

        const minX = NavArrows.MARGIN, maxX = cam.width - NavArrows.MARGIN;
        const minY = NavArrows.MARGIN, maxY = cam.height - NavArrows.MARGIN;
        const ox = toScreenX(me.x), oy = toScreenY(me.y);

        const seen = {};
        for (const id in scene.players) {
            const target = scene.players[id];
            if (!target || target.isCorpse || id === scene.myClientId) {
                continue;
            }

            // On-screen targets need no arrow — the cultist can already see them.
            if (view.contains(target.x, target.y)) {
                if (this.arrows[id]) this.arrows[id].setVisible(false);
                continue;
            }

            const dx = toScreenX(target.x) - ox;
            const dy = toScreenY(target.y) - oy;
            if (dx === 0 && dy === 0) {
                continue;
            }

            // Clamp the ray from the player to the inset viewport rectangle.
            let t = Infinity;
            if (dx > 0) t = Math.min(t, (maxX - ox) / dx);
            else if (dx < 0) t = Math.min(t, (minX - ox) / dx);
            if (dy > 0) t = Math.min(t, (maxY - oy) / dy);
            else if (dy < 0) t = Math.min(t, (minY - oy) / dy);

            const arrow = this._getArrow(id);
            arrow.setPosition(ox + dx * t, oy + dy * t);
            arrow.setRotation(Math.atan2(dy, dx));

            const isAlly = scene.cultistIds && scene.cultistIds.includes(id);
            const tint = isAlly ? NavArrows.ALLY_COLOR
                : (Number.isFinite(target.initialTint) ? target.initialTint : 0xffffff);
            arrow.setTint(tint);
            arrow.setVisible(true);
            seen[id] = true;
        }

        // Drop arrows for players that have left.
        for (const id in this.arrows) {
            if (!seen[id] && !scene.players[id]) {
                this.arrows[id].destroy();
                delete this.arrows[id];
            }
        }
    }
}
