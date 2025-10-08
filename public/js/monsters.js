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

    constructor (scene, statData, spriteKey, frame)
    {
        super(scene, statData.x, statData.y, spriteKey, frame);

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
        this.hpText = this.scene.add.text(statData.x, statData.y, statData.hp + '/' + this.maxHp, { font: '14px Arial', fill: '#ffffff' })
            .setOrigin(0.5, 1)
            .setDepth(DEPTH_MONSTER + 1);

        this.scene.add.existing(this);
        this.scene.physics.add.existing(this);

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
            this.setTint(0x333333);
            // avoid late changes of damage effect
            this.scene.time.delayedCall(110, () => this.setTint(0x333333), [], this);
            this.setDepth(DEPTH_DEAD);
            this.disableBody();
        }
    }

    updatePosition(posData)
    {
        if (this.isCorpse) {
            return;
        }

        this.x = posData.x;
        this.y = posData.y;

        if (posData.isAttacking && !this.isAttacking) {
            if (this.kind === 'demon') {
                this.anims.play('demon_attack', true);
            } else {
                // example of attack effect
                this.setTint(0x00ff00);
            }

            this.isAttacking = true;
        } else if (!posData.isAttacking && this.isAttacking) {
            if (this.kind === 'demon') {
                this.anims.play('demon', true);
            } else {
                this.clearTint()
            }
            this.isAttacking = false;
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
    kind = 'archer';
    bowSprite;

    constructor (scene, statData)
    {
        super(scene, statData, 'archer', 0);
    }

    preUpdate(time, delta)
    {
        super.preUpdate(time, delta);

        if (this.bowSprite) {
            this.bowSprite.x = this.x;
            this.bowSprite.y = this.y;
        }
    }

    spawn(statData)
    {
        super.spawn(statData);

        this.bowSprite = this.scene.add.sprite(statData.x, statData.y, 'bow', 0)
            .setScale(2)
            .setDepth(DEPTH_MONSTER + 0.1) // above body
            .setOrigin(0.5, 0.5);
    }
}

class Skeleton extends Monster
{
    kind = 'skeleton';

    constructor (scene, statData)
    {
        super(scene, statData, 'skeleton', 0);
    }
}

class Demon extends Monster
{
    kind = 'demon';

    constructor (scene, statData)
    {
        super(scene, statData, 'demon', 0);
        this.anims.play('demon', true);
    }
}
