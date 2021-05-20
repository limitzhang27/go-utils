package skiplist_demo

import "math/rand"

const (
    maxLevel int = 6 // should be enough for 2^16 elements
    p float32    = 0.25 //
)

// Element is an Element of a skiplist.
type Element struct {
    Score   float64 // 可排序键
    Value   interface{} // 键对应的值
    forward []*Element  // 向前的指针列表
}

func newElement(score float64, value interface{}, level int) *Element {
    return &Element{
        Score: score,
        Value: value,
        forward: make([]*Element, level),
    }
}

// SkipList represents a skiplist.
// The zero value from skiplist is an empty skiplist ready to use.
type SkipList struct {
    header *Element // header is a dummy element.
    len int         // current skiplist length, header not include
    level int       // current skiplist level, header not include
}

// New returns a new empty SkipList.
func New() *SkipList {
    return &SkipList{
        header: &Element{forward: make([]*Element, maxLevel)},
    }
}


// 随机生成level
func randomLevel() int {
    level := 1
    for rand.Float32() < p && level < maxLevel {
        level++
    }
    return level
}


// Front returns first element in the skiplist which maybe nil.
func (sl *SkipList) Front() *Element {
    return sl.header.forward[0]
}

func (e *Element) Next() *Element {
    if e != nil {
        return e.forward[0]
    }
    return nil
}

// Search the skiplist to findout element with the given score.
func (sl *SkipList) Search(score float64) (element *Element, ok bool) {
    x := sl.header
    for i := sl.level - 1; i >= 0; i-- {
        for x.forward[i] != nil && x.forward[i].Score < score {
            x = x.forward[i]
        }
    }

    x = x.forward[0]
    if x != nil && x.Score == score {
        return x, true
    }
    return nil, false
}

func (sl *SkipList) Insert(score float64, value interface{}) *Element {
    update := make([]*Element, maxLevel)
    x := sl.header
    for i := sl.level; i >= 0; i-- {
        for x.forward[i] != nil && x.forward[i].Score < score {
            x = x.forward[i]
        }
        update[i] = x
    }
    x = x.forward[0]

    // Score already presents, replace with new value then return.
    if x != nil && x.Score == score {
        x.Value = value
        return x
    }

    level := randomLevel()
    if level > sl.level {
        level = sl.level + 1
        update[sl.level] = sl.header
        sl.level = level
    }

    e := newElement(score, value, level)
    for i := 0; i < level; i++ {
        e.forward[i] = update[i].forward[i]
        update[i].forward[i] = e
    }
    sl.level++
    return e
}

func (sl *SkipList) Delete(score float64) *Element {
    update := make([]*Element, maxLevel)
    x := sl.header
    for i := sl.level - 1; i >= 0; i-- {
        for x.forward[i] != nil && x.forward[i].Score < score {
            x = x.forward[i]
        }
        update[i] = x
    }
    x = x.forward[0]
    if x != nil && x.Score == score {
        for i := 0; i < sl.level; i++ {
            if update[i].forward[i] != x {
                return nil
            }
            update[i].forward[i] = x.forward[i]
        }
        sl.len--
    }
    return x
}
