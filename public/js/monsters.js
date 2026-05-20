class Monster extends Phaser.Physics.Arcade.Sprite
{
    id;
    kind;
    hp = 100;
    hpText;
    isAttacking = false;
    scene;
    isCorpse = false;

    constructor (kind, scene, statData, spriteKey, frame, scale)
    {
        super(scene, statData.x, statData.y, spriteKey, frame);

        this.kind = kind;
        this.scene = scene;
        this.spawn(statData, scale);
    }

    spawn(statData, scale)
    {
        this.x = statData.x;
        this.y = statData.y;
        this.setScale(scale);
        this.setDepth(DEPTH_MONSTER);

        this.id = statData.id;
        this.hp = statData.hp;
        this.hpText = this.scene.add.text(statData.x, statData.y, statData.hp + '/' + statData.maxHp, { font: '10px Arial', fill: '#ffffff' })
            .setOrigin(0.5, 1)
            .setDepth(DEPTH_MONSTER + 1)
            .setMask(this.scene.monsterMask);

        this.scene.add.existing(this);
        this.scene.physics.add.existing(this);
        this.setMask(this.scene.monsterMask);

        if (this.hp <= 0) {
            this.isCorpse = true;
            this.convertToCorpse();
        }
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
            this.setTint(0x333333);
            // avoid late changes of damage effect
            this.scene.time.delayedCall(110, () => this.setTint(0x333333), [], this);
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
            this.playAttackAnimation(posData);
            return;
        }

        if (posData.isMoving) {
            this.playMoveAnimation(posData);
            return;
        }

        this.playIdleAnimation(posData);
    }

    playAttackAnimation(posData)
    {
        if (posData.direction === 'left') {
            this.setFlipX(true);
        } else if (posData.direction === 'right') {
            this.setFlipX(false);
        }

        let attackAnimsKey = `${this.kind}_attack_${posData.direction}`;
        if (!this.scene.anims.exists(attackAnimsKey)) {
            attackAnimsKey = `${this.kind}_attack`;
        }

        if (this.scene.anims.exists(attackAnimsKey)) {
            this.anims.play(attackAnimsKey, true);
        } else {
            console.warn("missing attack anims:", attackAnimsKey);
        }
    }

    playMoveAnimation(posData)
    {
        if (posData.direction === 'left') {
            this.setFlipX(true);
        } else if (posData.direction === 'right') {
            this.setFlipX(false);
        }

        let moveAnimsKey = `${this.kind}_walk_${posData.direction}`;
        if (!this.scene.anims.exists(moveAnimsKey)) {
            moveAnimsKey = `${this.kind}_walk`;
        }
        if (!this.scene.anims.exists(moveAnimsKey)) {
            moveAnimsKey = this.kind;
        }

        if (this.scene.anims.exists(moveAnimsKey)) {
            this.anims.play(moveAnimsKey, true);
        } else {
            console.warn("missing move anims:", moveAnimsKey);
        }
    }

    playIdleAnimation(posData)
    {
        let idleAnimsKey = `${this.kind}_idle_${posData.direction}`;
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
        this.scene.time.delayedCall(100, () => this.clearTint(), [], this);
    }

    static SpawnNewMonster(scene, statData)
    {
        switch (statData.kind) {
            case 'archer': return new Archer(scene, statData);
            case 'skeleton': return new Skeleton(scene, statData);
            case 'demon': return new Demon(scene, statData);
            case 'golem': return new Golem(scene, statData);
            case 'spider': return new Spider(scene, statData);
            case 'jelly': return new Jelly(scene, statData);
            case 'jelly_small': return new JellySmall(scene, statData);
            case 'jelly_micro': return new JellyMicro(scene, statData);
            default:
                console.error('Unknown monster kind:', statData.kind);
                return null;
        }
    }
}

class Archer extends Monster
{
    bowSprite;

    constructor (scene, statData)
    {
        super('archer', scene, statData, 'archer', 0, 2);

        if (this.isCorpse) {
            return;
        }

        this.bowSprite = this.scene.add.sprite(statData.x , statData.y + 10, 'bow', 6)
            .setScale(1.5)
            .setDepth(DEPTH_MONSTER + 0.1) // above body
            .setOrigin(0.5, 0.5)
            .setMask(this.scene.monsterMask);
    }

    preUpdate(time, delta)
    {
        super.preUpdate(time, delta);

        if (this.bowSprite) {
            this.bowSprite.x = this.x;
            this.bowSprite.y = this.y + 15;
        }
    }

    playAttackAnimation(posData)
    {
        // super.playAttackAnimation(posData);
        this.bowSprite.anims.play(`bow_${posData.direction}`, true);
    }

    convertToCorpse()
    {
        super.convertToCorpse();
        if (this.bowSprite) {
            this.bowSprite.destroy();
            this.bowSprite = null;
        }
    }
}

class Skeleton extends Monster
{
    constructor (scene, statData)
    {
        super('skeleton', scene, statData, 'skeleton', 0, 0.8);
    }
}

class Demon extends Monster
{
    constructor (scene, statData)
    {
        super('demon', scene, statData, 'demon', 0, 2);
        this.anims.play('demon', true);
    }
}

class Spider extends Monster
{
    _wasAttacking = false;

    constructor (scene, statData)
    {
        super('spider', scene, statData, 'spider', 0, 2);
        this.anims.play('spider', true);
    }

    playAttackAnimation(posData)
    {
        if (posData.direction === 'left') this.setFlipX(true);
        else if (posData.direction === 'right') this.setFlipX(false);

        if (!this._wasAttacking) {
            this._wasAttacking = true;
            this.anims.play('spider_attack', false);
        }
    }

    playMoveAnimation(posData)
    {
        this._wasAttacking = false;
        if (posData.direction === 'left') this.setFlipX(true);
        else if (posData.direction === 'right') this.setFlipX(false);
        this.anims.play('spider', true);
    }

    playIdleAnimation(posData)
    {
        this._wasAttacking = false;
        this.anims.play('spider', true);
    }
}

class Golem extends Monster
{
    _wasAttacking = false;

    constructor (scene, statData)
    {
        super('golem', scene, statData, 'golem', 0, 2);
        this.anims.play('golem', true);
    }

    playAttackAnimation(posData)
    {
        if (posData.direction === 'left') this.setFlipX(true);
        else if (posData.direction === 'right') this.setFlipX(false);

        if (!this._wasAttacking) {
            this._wasAttacking = true;
            this.anims.play('golem_attack', false);
        }
    }

    playMoveAnimation(posData)
    {
        this._wasAttacking = false;
        super.playMoveAnimation(posData);
    }

    playIdleAnimation(posData)
    {
        this._wasAttacking = false;
        super.playIdleAnimation(posData);
    }
}

class Jelly extends Monster
{
    _wasAttacking = false;
    _isSplitting = false;

    constructor (scene, statData, kind = 'jelly', scale = 1)
    {
        super(kind, scene, statData, 'jelly', 26, scale);
        if (!this.isCorpse) {
            this.anims.play('jelly', true);
        }
    }

    triggerSplit(onComplete)
    {
        this._isSplitting = true;
        this._splitOnComplete = onComplete;
    }

    convertToCorpse()
    {
        if (this.hpText) {
            this.hpText.destroy();
            this.hpText = null;
        }
        this.setDepth(DEPTH_DEAD);
        this.disableBody();

        if (this._isSplitting) {
            this.anims.play('split', false);
            this.once('animationcomplete', () => {
                this.destroy();
                if (this._splitOnComplete) this._splitOnComplete();
            });
        } else {
            this.anims.play('jelly_dead', false);
        }
    }

    playAttackAnimation(posData)
    {
        if (posData.direction === 'left') this.setFlipX(true);
        else if (posData.direction === 'right') this.setFlipX(false);
        if (!this._wasAttacking) {
            this._wasAttacking = true;
            this.anims.play('jelly_attack', false);
        }
    }

    playMoveAnimation(posData)
    {
        this._wasAttacking = false;
        if (posData.direction === 'left') this.setFlipX(true);
        else if (posData.direction === 'right') this.setFlipX(false);
        this.anims.play('jelly', true);
    }

    playIdleAnimation()
    {
        this._wasAttacking = false;
        this.anims.play('jelly', true);
    }
}

class JellySmall extends Jelly
{
    constructor (scene, statData)
    {
        super(scene, statData, 'jelly_small', 0.5);
    }
}

class JellyMicro extends Jelly
{
    constructor (scene, statData)
    {
        super(scene, statData, 'jelly_micro', 0.3);
    }

    convertToCorpse()
    {
        if (this.hpText) {
            this.hpText.destroy();
            this.hpText = null;
        }
        this.setDepth(DEPTH_DEAD);
        this.disableBody();
        this.anims.play('jelly_dead', false);
    }
}
