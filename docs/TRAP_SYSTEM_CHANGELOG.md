# Changelog: Trap System FSM Implementation

## История изменений

### v2.0 (25 ноября 2025) - Процентная система времени
**Упрощение настройки через процентное распределение периода**

#### Изменения в параметрах:
- **Было (абсолютные времена):**
  - `warningTime`, `activeTime`, `cooldownTime` в секундах
  - FSM: Armed → Warning → Active → Cooldown → Armed
  
- **Стало (проценты от периода):**
  - `period` - общее время цикла (сек)
  - `phase` - смещение старта (сек)
  - `activePercent` - % времени на активную фазу (по умолчанию 10%)
  - `cooldownPercent` - % времени на восстановление (по умолчанию 20%)
  - `armedPercent` - автоматически = 100% - active% - cooldown%
  - FSM: Armed → **Active** → Cooldown → Armed (удалено состояние Warning)

#### Новая логика Active фазы:
- **Первые 40%** времени Active: кадры 7-11 (анимация подъема шипов)
- **Оставшиеся 60%**: кадр 0 (полностью выдвинутые шипы)
- **Урон наносится всегда** в течение всего состояния Active

#### Примеры конфигураций:

**Быстрые опасные шипы:**
```json
{
  "activator": "timer",
  "period": 2.0,
  "activePercent": 15,
  "cooldownPercent": 25,
  "damage": 25
}
```
Результат: 60% armed (1.2с), 15% active (0.3с), 25% cooldown (0.5с)

**Медленные шипы с долгим ожиданием:**
```json
{
  "activator": "timer",
  "period": 6.0,
  "activePercent": 8,
  "cooldownPercent": 12,
  "damage": 30
}
```
Результат: 80% armed (4.8с), 8% active (0.48с), 12% cooldown (0.72с)

**Автоматическая нормализация:** если `activePercent + cooldownPercent > 100%`, система пропорционально уменьшает их.

---

### v1.0 (24 ноября 2025) - Первоначальная FSM реализация

#### ❌ Удалено / Deprecated
- **Старая клиентская логика урона от шипов** (`game_event_handler.js`): 
  - Убрана проверка `safeFrames` на клиенте
  - Убрана отправка команды `HitPlayerCommand` с клиента
  - Теперь урон **полностью контролируется сервером**

#### ✅ Добавлено

**Backend (Go):**
1. **Новый файл:** `internal/game/traps.go`
   - FSM с состояниями: Disabled, Armed, Active, Cooldown
   - Типы активации: Timer, Link
   - Параметры: ActivePercent, CooldownPercent, Damage

2. **Изменения в `game.go`:**
   - Добавлено поле `traps map[string]*Trap`
   - Создание трапов при загрузке карты из объектов `trap_spikes`
   - Отправка начального состояния трапов клиентам
   - Парсинг процентных параметров из Tiled

3. **Изменения в `game_objects.go`:**
   - Функция `tickTraps()` - обновление состояний всех трапов
   - Функция `checkTrapDamage()` - серверная проверка урона
   - Триггеры активируют трапы через `Activate()`

4. **Новое событие:** `TrapStateChangedEvent` в `events.go`

**Frontend (JavaScript):**
1. **Изменения в `game_event_handler.js`:**
   - Обработчик `TrapStateChangedEvent`
   - Функция `createTrapSprite()`
   - Color-coded debug rectangles (зеленый=armed, красный=active, синий=cooldown)

2. **Изменения в `scene.game.js`:**
   - Поле `traps = {}`
   - Инициализация трапов из `gameData.traps`

#### 🔄 Обратная совместимость
- Старые карты с `SpawnSpikeEvent` продолжают работать
- Поле `spikeEvents` оставлено для совместимости

---

## Кадры анимации (текущий маппинг)

- **Кадр 0:** Active peak (полностью выдвинутые, урон)
- **Кадры 1-4:** Cooldown (опускаются)
- **Кадр 5:** Armed (спрятаны)
- **Кадры 7-11:** Active start (поднимаются, урон уже есть!)
- **~~Кадр 6:~~** Удален (больше не используется)

## Debug цвета

- **🟢 Зеленый:** Armed (безопасно)
- **🔴 Красный:** Active (опасно!)
- **🔵 Синий:** Cooldown (восстановление)

## Как проверить

1. Запустите сервер: `go run cmd/dungeon.go`
2. Откройте карту в Tiled
3. Добавьте объект `trap_spikes` с properties:
   ```
   activator = "timer"
   period = 4.0
   activePercent = 10
   cooldownPercent = 20
   ```
4. Зайдите в игру и наблюдайте:
   - Зеленый прямоугольник в состоянии Armed (2.8 сек)
   - Красный прямоугольник + анимация подъема (7-11) → кадр 0 (0.4 сек)
   - Синий прямоугольник + анимация опускания (1-4) (0.8 сек)
   - Урон при наступании на ловушку в Active фазе

## Документация
См. `docs/TRAPS_USAGE.md` для подробных инструкций по настройке в Tiled.
