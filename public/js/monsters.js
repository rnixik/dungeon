class Monster extends Phaser.Physics.Arcade.Sprite
{
    id;
    kind;
    maxHp = 100;
    hp = 100;
    hpText;
    isAttacking = false;
    scene;
    isCorpse = false;

    constructor (kind, scene, statData, spriteKey, frame)
    {
        super(scene, statData.x, statData.y, spriteKey, frame);

        this.kind = kind;
        this.scene = scene;
        this.spawn(statData);
    }

    spawn(statData)
    {
        this.x = statData.x;
        this.y = statData.y;
        this.setScale(2);
        this.setDepth(DEPTH_MONSTER);

        this.id = statData.id;
        this.hp = statData.hp;
        this.hpText = this.scene.add.text(statData.x, statData.y, statData.hp + '/' + this.maxHp, { font: '10px Arial', fill: '#ffffff' })
            .setOrigin(0.5, 1)
            .setDepth(DEPTH_MONSTER + 1)
            .setMask(this.scene.mask);

        this.scene.add.existing(this);
        this.scene.physics.add.existing(this);
        this.setMask(this.scene.mask);

        if (this.hp <= 0) {
            this.isCorpse = true;
            this.convertToCorpse();
        }
    }

    updateStatAndPosition(statData)
    {
        if (this.hp !== statData.hp) {
            this.hp = statData.hp;
            if (this.hpText) {
                this.hpText.setText(statData.hp + '/' + this.maxHp);
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

    static SpawnNewMonster(scene, statData)
    {
        let monster;
        if (statData.kind === 'archer') {
            monster = new Archer(scene, statData);
        } else if (statData.kind === 'skeleton') {
            monster = new Skeleton(scene, statData);
        } else if (statData.kind === 'demon') {
            monster = new Demon(scene, statData);
        }  else {
            console.error('Unknown monster kind:', statData.kind);
            return null;
        }

        return monster;
    }
}

class Archer extends Monster
{
    bowSprite;

    constructor (scene, statData)
    {
        super('archer', scene, statData, 'archer', 0);

        if (this.isCorpse) {
            return;
        }

        this.bowSprite = this.scene.add.sprite(statData.x , statData.y + 10, 'bow', 6)
            .setScale(1.5)
            .setDepth(DEPTH_MONSTER + 0.1) // above body
            .setOrigin(0.5, 0.5)
            .setMask(this.scene.mask);
    }

    preUpdate(time, delta)
    {
        super.preUpdate(time, delta);

        if (this.bowSprite) {
            this.bowSprite.x = this.x;
            this.bowSprite.y = this.y;
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
        super('skeleton', scene, statData, 'skeleton', 0);
    }
}

class Demon extends Monster
{
    constructor (scene, statData)
    {
        super('demon', scene, statData, 'demon', 0);
        this.anims.play('demon', true);
    }
}
