class Player extends Phaser.Physics.Arcade.Sprite
{
    id;
    kind;
    hp = 100;
    hpText;
    isAttacking = false;
    scene;
    isCorpse = false;
    initialTint = 0xffffff;

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
        this.setScale(PLAYER_SCALE);
        this.setDepth(DEPTH_PLAYER);
        this.setMask(this.scene.mask);

        if (this.hp <= 0) {
            this.isCorpse = true;
            this.convertToCorpse();

            return;
        }

        this.initialTint = Number(statData.color)
        this.setTint(this.initialTint);

        this.id = statData.clientId;
        this.hp = statData.hp;
        this.hpText = this.scene.add.text(statData.x, statData.y, statData.hp + '/' + statData.maxHp, { font: '10px Arial', fill: '#ffffff' })
            .setOrigin(0.5, 1)
            .setDepth(DEPTH_PLAYER + 1)
            .setMask(this.scene.mask);

        this.scene.add.existing(this);
        this.scene.physics.add.existing(this);
        this.body.setSize(30, 20).setOffset(0, 10);
    }

    preUpdate(time, delta)
    {
        super.preUpdate(time, delta);

        if (this.hpText) {
            this.hpText.x = this.x;
            this.hpText.y = this.y - 20;
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

        this.updatePosition(statData);
    }

    convertToCorpse()
    {
        if (this.hpText) {
            this.hpText.destroy();
        }

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
                this.setAngle(0).setFlipX(true);
            } else if (direction === 'right') {
                this.setAngle(0).setFlipX(false);
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
            this.setAngle(0).setFlipX(true);
        } else if (direction === 'right') {
            this.setAngle(0).setFlipX(false);
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

    takeDamage(damage)
    {
        if (this.isCorpse) {
            return;
        }

        this.setTint(0xff3333);
        this.scene.time.delayedCall(100, () => this.setTint(this.initialTint), [], this);
    }
}

class MyPlayer extends Player
{
    updateStatAndPosition(statData)
    {
        if (this.hp !== statData.hp) {
            this.hp = statData.hp;
            if (this.hpText) {
                this.hpText.setText(statData.hp + '/' + statData.maxHp);
            }
        }
    }

    shouldStopMovement()
    {
        if (this.kind !== 'knight') {
            return false;
        }

        return this.isAttacking && this.anims.currentFrame.index >= 4;
    }
}