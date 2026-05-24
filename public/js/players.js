class Player extends Phaser.Physics.Arcade.Sprite
{
    id;
    kind;
    hp = 100;
    hpText;
    avatarImage = null;
    isAttacking = false;
    scene;
    isCorpse = false;
    initialTint = 0xffffff;
    _protectionGraphics = null;
    _speedBoostGraphics = null;

    constructor (kind, scene, statData)
    {
        super(scene, statData.x, statData.y, kind, 1);

        this.kind = kind;
        this.scene = scene;
        this.spawn(statData);
    }

    spawn(statData)
    {
        this.x = statData.x;
        this.y = statData.y;
        this.setOrigin(0.5, 0.5);
        this.setScale(PLAYER_SCALE);
        this.setDepth(DEPTH_PLAYER);
        this.setMask(this.scene.mask);

        if (this.hp <= 0) {
            this.isCorpse = true;
            this.convertToCorpse();

            return;
        }

        this.initialTint = Number(statData.color)
        this.setTint(0xffffff);

        this.id = statData.clientId;
        this.hp = statData.hp;
        this.hpText = this.scene.add.text(statData.x, statData.y, statData.hp + '/' + statData.maxHp, { font: '10px Arial', fill: '#ffffff' })
            .setOrigin(0.5, 1)
            .setDepth(DEPTH_PLAYER + 1)
            .setMask(this.scene.mask);

        if (statData.avatarUrl) {
            this._loadAndShowAvatar(statData.avatarUrl);
        }

        this.scene.add.existing(this);
        this.scene.physics.add.existing(this);
        this.body.setSize(64, 34).setOffset(0, 30);
    }

    preUpdate(time, delta)
    {
        super.preUpdate(time, delta);

        if (!this.isCorpse) {
            this.setDepth(DEPTH_PLAYER + this.y * 0.01);
        }

        if (this.hpText) {
            this.hpText.x = this.x;
            this.hpText.y = this.y - 30;
            this.hpText.setDepth(this.depth + 1);
        }
        if (this.avatarImage) {
            this.avatarImage.x = this.x - (this.hpText ? this.hpText.width / 2 + 12 : 12);
            this.avatarImage.y = this.y - 30;
            this.avatarImage.setDepth(this.depth + 1);
        }
        if (this._protectionGraphics) {
            this._protectionGraphics.x = this.x;
            this._protectionGraphics.y = this.y;
            this._protectionGraphics.setDepth(this.depth + 0.5);
        }
        if (this._speedBoostGraphics) {
            this._speedBoostGraphics.x = this.x;
            this._speedBoostGraphics.y = this.y;
            this._speedBoostGraphics.setDepth(this.depth + 0.5);
        }
    }

    updateStatAndPosition(statData)
    {
        if (this.hp !== statData.hp) {
            this.hp = statData.hp;
            if (this.hpText) {
                this.hpText.setText(statData.hp + '/' + statData.maxHp);
            }
        }

        if (this.hp <= 0 && !this.isCorpse) {
            this.isCorpse = true;
            this.convertToCorpse();
        }

        if (statData.avatarUrl && !this.avatarImage && !this.isCorpse) {
            this._loadAndShowAvatar(statData.avatarUrl);
        }

        if (statData.speedBoostPercent > 0) {
            this.showSpeedBoost();
        } else {
            this.hideSpeedBoost();
        }

        if (statData.hasShield) {
            this.showProtection();
        } else {
            this.hideProtection();
        }

        this.updatePosition(statData);
    }

    convertToCorpse()
    {
        if (this.hpText) {
            this.hpText.destroy();
            this.hpText = null;
        }
        if (this.avatarImage) {
            this.avatarImage.destroy();
            this.avatarImage = null;
        }
        this.hideProtection();
        this.hideSpeedBoost();

        this.setDepth(DEPTH_DEAD);
        this.disableBody();

        const animsKey = `${this.kind}_dead`;
        if (this.scene.anims.exists(animsKey)) {
            this.anims.play(animsKey, true);
        } else {
            console.warn("no death anims:", animsKey);
            this.anims.stop();
            this.setTint(0xff3333);
            // avoid late changes of damage effect
            this.scene.time.delayedCall(110, () => this.setTint(0xff3333), [], this);
        }
    }

    respawn(x, y)
    {
        this.isCorpse = false;
        this.enableBody(true, x, y, true, true);
        this.setDepth(DEPTH_PLAYER);
        this.setTint(0xffffff);
    }

    updatePosition(posData)
    {
        if (this.isCorpse) {
            return;
        }

        this.x = posData.x;
        this.y = posData.y;

        if (posData.isAttacking) {
            this.isAttacking = true;
            this.playAttackAnimation(posData.direction);
            return;
        }
        this.isAttacking = false;

        if (posData.isMoving) {
            this.playMoveAnimation(posData.direction);
            return;
        }

        this.playIdleAnimation(posData.direction);
    }

    playAttackAnimation(direction)
    {
        let attackAnimsKey = `${this.kind}_attack_${direction}`;
        if (!this.scene.anims.exists(attackAnimsKey)) {
            attackAnimsKey = `${this.kind}_attack`;

            if (direction === 'left') {
                this.setFlipX(true);
            } else if (direction === 'right') {
                this.setFlipX(false);
            }
        }

        if (this.scene.anims.exists(attackAnimsKey)) {
            this.anims.play(attackAnimsKey, true);
        } else {
            console.warn("missing attack anims:", attackAnimsKey);
        }
    }

    playMoveAnimation(direction)
    {
        if (this.isCorpse) {
            return;
        }

        if (direction === 'left') {
            this.setFlipX(true);
        } else if (direction === 'right') {
            this.setFlipX(false);
        }

        let moveAnimsKey = `${this.kind}_walk_${direction}`;
        if (!this.scene.anims.exists(moveAnimsKey)) {
            moveAnimsKey = `${this.kind}_walk`;
        }
        if (!this.scene.anims.exists(moveAnimsKey)) {
            moveAnimsKey = this.kind;
        }

        if (!this.scene.anims.exists(moveAnimsKey)) {
            moveAnimsKey = `${this.kind}_idle`;
        }

        if (this.scene.anims.exists(moveAnimsKey)) {
            this.anims.play(moveAnimsKey, true);
        } else {
            console.warn("missing move anims:", this.kind);
        }
    }

    playIdleAnimation(direction)
    {
        if (this.isCorpse) {
            return;
        }

        // Handle sprite flipping for idle animation
        if (direction === 'left') {
            this.setFlipX(true);
        } else if (direction === 'right') {
            this.setFlipX(false);
        }

        let idleAnimsKey = `${this.kind}_idle_${direction}`;
        if (!this.scene.anims.exists(idleAnimsKey)) {
            idleAnimsKey = this.kind + '_idle';
        }
        if (!this.scene.anims.exists(idleAnimsKey)) {
            idleAnimsKey = this.kind;
        }
        if (this.scene.anims.exists(idleAnimsKey)) {
            this.anims.play(idleAnimsKey, true);
        }
    }

    showProtection()
    {
        if (this._protectionGraphics) {
            return;
        }
        const g = this.scene.add.graphics();
        g.lineStyle(3, 0x4499ff, 0.85);
        g.strokeCircle(0, 0, 27);
        g.lineStyle(1, 0xaaddff, 0.4);
        g.strokeCircle(0, 0, 31);
        g.setDepth(DEPTH_PLAYER + 0.5);
        g.setMask(this.scene.mask);
        this._protectionGraphics = g;
    }

    hideProtection()
    {
        if (this._protectionGraphics) {
            this._protectionGraphics.destroy();
            this._protectionGraphics = null;
        }
    }

    showSpeedBoost()
    {
        if (this._speedBoostGraphics) {
            return;
        }
        const g = this.scene.add.graphics();
        g.lineStyle(3, 0xffcc00, 0.85);
        g.strokeCircle(0, 0, 17);
        g.lineStyle(1, 0xffee88, 0.4);
        g.strokeCircle(0, 0, 21);
        g.setDepth(DEPTH_PLAYER + 0.5);
        g.setMask(this.scene.mask);
        this._speedBoostGraphics = g;
    }

    hideSpeedBoost()
    {
        if (this._speedBoostGraphics) {
            this._speedBoostGraphics.destroy();
            this._speedBoostGraphics = null;
        }
    }

    _loadAndShowAvatar(url)
    {
        const key = 'avatar_' + this.id;
        if (this.scene.textures.exists(key)) {
            this._createAvatarImage(key);
            return;
        }
        const img = new Image();
        img.crossOrigin = 'anonymous';
        img.onload = () => {
            if (!this.scene || !this.scene.textures || this.isCorpse) return;
            const size = 20;
            const canvas = document.createElement('canvas');
            canvas.width = size;
            canvas.height = size;
            const ctx = canvas.getContext('2d');
            ctx.beginPath();
            ctx.arc(size / 2, size / 2, size / 2, 0, Math.PI * 2);
            ctx.closePath();
            ctx.clip();
            ctx.drawImage(img, 0, 0, size, size);
            this.scene.textures.addCanvas(key, canvas);
            this._createAvatarImage(key);
        };
        img.onerror = () => {};
        img.src = url;
    }

    _createAvatarImage(key)
    {
        if (this.avatarImage) this.avatarImage.destroy();
        this.avatarImage = this.scene.add.image(this.x, this.y - 30, key)
            .setOrigin(0.5, 1)
            .setDepth(DEPTH_PLAYER + 1)
            .setMask(this.scene.mask);
    }

    setDisplayAlpha(alpha)
    {
        this.setAlpha(alpha);
        if (this.hpText) this.hpText.setAlpha(alpha);
        if (this.avatarImage) this.avatarImage.setAlpha(alpha);
        if (this._protectionGraphics) this._protectionGraphics.setAlpha(alpha);
        if (this._speedBoostGraphics) this._speedBoostGraphics.setAlpha(alpha);
    }

    takeDamage(damage)
    {
        if (this.isCorpse) {
            return;
        }

        this.setTint(0xff3333);
        this.scene.time.delayedCall(150, () => this.setTint(0xffffff), [], this);
    }
}

class MyPlayer extends Player
{
    speedBoostPercent = 0;
    webSlowMultiplier = 1;
    jellyHitSlowUntil = 0;
    jellyAuraSlow = false;

    updateStatAndPosition(statData)
    {
        if (this.hp !== statData.hp) {
            this.hp = statData.hp;
            if (this.hpText) {
                this.hpText.setText(statData.hp + '/' + statData.maxHp);
            }
        }
        if (statData.speedBoostPercent !== undefined) {
            this.speedBoostPercent = statData.speedBoostPercent;
        }

        if (this.speedBoostPercent > 0) {
            this.showSpeedBoost();
        } else {
            this.hideSpeedBoost();
        }

        if (statData.hasShield) {
            this.showProtection();
        } else {
            this.hideProtection();
        }
    }

    shouldStopMovement()
    {
        if (this.kind !== 'knight') {
            return false;
        }

        return this.isAttacking && this.anims.currentFrame.index >= 4;
    }

    getMovementVelocity()
    {
        let velocity = 180;
        switch (this.kind) {
            case 'knight':
                if (this.isAttacking && this.anims.currentFrame.index >= 3) {
                    velocity = 0;
                }
                break;
            case 'rogue': {
                if (this.isAttacking) {
                    velocity = 220;
                }
                break;
            }
            case 'mage':
                if (this.isAttacking) {
                    velocity = 140;
                }
                break;
        }

        if (this.speedBoostPercent > 0 && velocity > 0) {
            velocity = Math.round(velocity * (1 + this.speedBoostPercent / 100));
        }

        let slowMultiplier = 1;
        if (this.webSlowMultiplier < slowMultiplier) slowMultiplier = this.webSlowMultiplier;
        if (this.jellyHitSlowUntil > Date.now()) slowMultiplier = Math.min(slowMultiplier, 0.2);
        if (this.jellyAuraSlow) slowMultiplier = Math.min(slowMultiplier, 0.8);
        if (slowMultiplier < 1 && velocity > 0) {
            velocity = Math.round(velocity * slowMultiplier);
        }

        return velocity;
    }
}