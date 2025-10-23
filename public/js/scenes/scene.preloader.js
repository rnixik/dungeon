var sceneConfigPreloader = {
    key: 'Preloader',
    preload: function() {
        this.cameras.main.backgroundColor = Phaser.Display.Color.HexStringToColor("#16181a");
        this.add.sprite(0, 0, 'main').setOrigin(0,0).setScale(2);

        const preloadBar = this.add.sprite(145, this.cameras.main.height - 165, 'preloaderBar').setOrigin(0,0);

        this.load.on('progress', function (value) {
            preloadBar.setCrop(0, 0, preloadBar.width * value, preloadBar.height);
        });

        this.load.image('tiles', 'assets/catacombs.png');

        this.load.spritesheet('mage', 'assets/MiniRouge\\3 - Heroes\\Hero 01 Mage\\32x32\\Hero01 Mage Idle2x-Sheet.png', { frameWidth: 32, frameHeight: 32 });
        //this.load.spritesheet('fireball', 'assets/MiniRouge\\3 - Heroes\\Hero 01 Mage\\32x32\\Fireball Magel-Sheet2x.png', { frameWidth: 32, frameHeight: 32 });
        this.load.spritesheet('fireball', 'assets/electrobolt.png', { frameWidth: 32, frameHeight: 32 });
        this.load.spritesheet('archer', 'assets/skeleton5.png', { frameWidth: 32, frameHeight: 32 });
        this.load.spritesheet('skeleton', 'assets/skeleton6.png', { frameWidth: 32, frameHeight: 32 });
        this.load.spritesheet('bow', 'assets/skeleton_bow.png', { frameWidth: 40, frameHeight: 40 });
        this.load.image('arrow', 'assets/arrow.png');
        this.load.image('mask', 'assets/mask1.png?v=2');
        this.load.atlas('controls', 'assets/controls.png', 'assets/controls.json');
        this.load.image('bullet', 'assets/bullet7.png');
        this.load.image('spinner', 'assets/spinner.png');
        //this.load.image('archer', 'assets/archer.png');
        //this.load.image('skeleton', 'assets/archer.png');

        this.load.spritesheet('demon', 'assets/demon/IDLE.png', { frameWidth: 79, frameHeight: 69 });
        this.load.spritesheet('demon_attack', 'assets/demon/ATTACK.png', { frameWidth: 79, frameHeight: 69 });

        this.load.spritesheet('chest', 'assets/chest.png', { frameWidth: 32, frameHeight: 32 });

        this.load.plugin('rexvirtualjoystickplugin', 'js/rexvirtualjoystickplugin.min.js', true);
    },

    create: function() {
        this.anims.create({
            key: 'mage_walk_left',
            frames: this.anims.generateFrameNumbers('mage', { start: 0, end: 11 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'mage_walk_right',
            frames: this.anims.generateFrameNumbers('mage', { start: 0, end: 11 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'mage_walk_up',
            frames: this.anims.generateFrameNumbers('mage', { start: 13, end: 17 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'mage_walk_down',
            frames: this.anims.generateFrameNumbers('mage', { start: 24, end: 35 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'mage_idle',
            frames: this.anims.generateFrameNumbers('mage', { start: 1, end: 11 }),
            frameRate: 5,
            repeat: -1
        });
        this.anims.create({
            key: 'fireball-loop',
            frames: this.anims.generateFrameNumbers('fireball', { start: 0, end: 4 }),
            frameRate: 10,
            repeat: -1
        });
        this.anims.create({
            key: 'demon',
            frames: this.anims.generateFrameNumbers('demon', { start: 0, end: 3 }),
            frameRate: 5,
            repeat: -1
        });
        this.anims.create({
            key: 'demon_attack',
            frames: this.anims.generateFrameNumbers('demon_attack', { start: 0, end: 7 }),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_idle_down',
            frames: this.anims.generateFrameNumbers('archer', { start: 0, end: 3}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_walk_down',
            frames: this.anims.generateFrameNumbers('archer', { start: 6, end: 11}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_idle_up',
            frames: this.anims.generateFrameNumbers('archer', { start: 12, end: 15}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_walk_up',
            frames: this.anims.generateFrameNumbers('archer', { start: 18, end: 23}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_idle_left',
            frames: this.anims.generateFrameNumbers('archer', { start: 24, end: 27}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_walk_left',
            frames: this.anims.generateFrameNumbers('archer', { start: 30, end: 35}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_idle_right',
            frames: this.anims.generateFrameNumbers('archer', { start: 36, end: 39}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_walk_right',
            frames: this.anims.generateFrameNumbers('archer', { start: 42, end: 47}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'archer_dead',
            frames: this.anims.generateFrameNumbers('archer', { start: 48, end: 53}),
            frameRate: 8,
            repeat: 0
        });
        this.anims.create({
            key: 'skeleton_idle_down',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 0, end: 3}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_walk_down',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 6, end: 11}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_idle_up',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 12, end: 15}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_walk_up',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 18, end: 23}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_idle_left',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 24, end: 27}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_walk_left',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 30, end: 35}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_idle_right',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 36, end: 39}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_walk_right',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 42, end: 47}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'skeleton_dead',
            frames: this.anims.generateFrameNumbers('skeleton', { start: 48, end: 53}),
            frameRate: 8,
            repeat: 0
        });
        this.anims.create({
            key: 'bow_up',
            frames: this.anims.generateFrameNumbers('bow', { start: 0, end: 5}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'bow_down',
            frames: this.anims.generateFrameNumbers('bow', { start: 6, end: 11}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'bow_right',
            frames: this.anims.generateFrameNumbers('bow', { start: 12, end: 17}),
            frameRate: 8,
            repeat: -1
        });
        this.anims.create({
            key: 'bow_left',
            frames: this.anims.generateFrameNumbers('bow', { start: 18, end: 23}),
            frameRate: 8,
            repeat: -1
        });

        this.scene.switch('MainMenu');
    }
};
