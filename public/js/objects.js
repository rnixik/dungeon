class GameObject extends Phaser.Physics.Arcade.Sprite
{
    id;
    kind;
    state;
    scene;

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
        this.setScale(1);
        this.setDepth(DEPTH_OBJECTS);

        this.id = statData.id;
        this.state = statData.scene;

        this.scene.add.existing(this);
        this.scene.physics.add.existing(this);
        this.body.setImmovable(true)
        this.setMask(this.scene.mask);
    }

    static SpawnNewObject(scene, statData)
    {
        switch (statData.kind) {
            case 'chest': return new Chest(scene, statData);
            default:
                console.error('Unknown object kind:', statData.kind);
                return null;
        }
    }
}

class Chest extends GameObject
{
    isOpen = false;

    constructor (scene, statData)
    {
        super('chest', scene, statData, 'chest', 0);
        if (statData.state === 'open') {
            this.setFrame(1);
            this.isOpen = true;
        }
    }

    open()
    {
        if (this.isOpen) {
            return;
        }
        this.isOpen = true;
        this.setFrame(1);
    }
}
