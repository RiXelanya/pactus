package committee

import (
	"container/list"
	"fmt"
	"sort"
	"strings"

	"github.com/pactus-project/pactus/crypto"
	"github.com/pactus-project/pactus/types/validator"
)

var _ Committee = &committee{}

type committee struct {
	committeeSize int
	validatorList *list.List
	proposerPos   *list.Element
}

func cloneValidator(val *validator.Validator) *validator.Validator {
	cloned := new(validator.Validator)
	*cloned = *val
	return cloned
}

func NewCommittee(validators []*validator.Validator, committeeSize int,
	proposerAddress crypto.Address,
) (Committee, error) {
	validatorList := list.New()
	var proposerPos *list.Element

	for _, val := range validators {
		el := validatorList.PushBack(cloneValidator(val))
		if val.Address().EqualsTo(proposerAddress) {
			proposerPos = el
		}
	}

	if proposerPos == nil {
		return nil, fmt.Errorf("Proposer is not in the list")
	}

	return &committee{
		committeeSize: committeeSize,
		validatorList: validatorList,
		proposerPos:   proposerPos,
	}, nil
}

func (c *committee) TotalPower() int64 {
	p := int64(0)
	c.iterate(func(v *validator.Validator) (stop bool) {
		p += v.Power()
		return false
	})
	return p
}

func (c *committee) Update(lastRound int16, joined []*validator.Validator) {
	sort.SliceStable(joined, func(i, j int) bool {
		return joined[i].Number() < joined[j].Number()
	})

	// First update the validator list
	for _, val := range joined {
		committeeVal := c.find(val.Address())
		if committeeVal == nil {
			c.validatorList.InsertBefore(cloneValidator(val), c.proposerPos)
		} else {
			committeeVal.UpdateLastSortitionHeight(val.LastSortitionHeight())

			// Ensure that a validator's stake and bonding properties
			// remain unchanged while they are part of the committee.
			// Refer to the Bond executor for additional details.
			if committeeVal.Stake() != val.Stake() ||
				committeeVal.LastBondingHeight() != val.LastBondingHeight() ||
				committeeVal.UnbondingHeight() != val.UnbondingHeight() {
				panic("validators within the committee must be consistent")
			}
		}
	}

	// Now, adjust the list
	oldestFirst := make([]*list.Element, c.validatorList.Len())
	i := 0
	for e := c.validatorList.Front(); e != nil; e = e.Next() {
		oldestFirst[i] = e
		i++
	}

	sort.SliceStable(oldestFirst, func(i, j int) bool {
		return oldestFirst[i].Value.(*validator.Validator).LastSortitionHeight() <
			oldestFirst[j].Value.(*validator.Validator).LastSortitionHeight()
	})

	for i := 0; i <= int(lastRound); i++ {
		c.proposerPos = c.proposerPos.Next()
		if c.proposerPos == nil {
			c.proposerPos = c.validatorList.Front()
		}
	}

	adjust := c.validatorList.Len() - c.committeeSize
	for i := 0; i < adjust; i++ {
		if oldestFirst[i] == c.proposerPos {
			c.proposerPos = c.proposerPos.Next()
			if c.proposerPos == nil {
				c.proposerPos = c.validatorList.Front()
			}
		}
		c.validatorList.Remove(oldestFirst[i])
	}
}

// Validators retrieves a list of all validators in the committee.
// A cloned instance of each validator is returned to avoid modification of the original objects.
func (c *committee) Validators() []*validator.Validator {
	vals := make([]*validator.Validator, c.validatorList.Len())
	i := 0
	c.iterate(func(v *validator.Validator) (stop bool) {
		vals[i] = cloneValidator(v)
		i++
		return false
	})

	return vals
}

func (c *committee) Contains(addr crypto.Address) bool {
	return c.find(addr) != nil
}

func (c *committee) find(addr crypto.Address) *validator.Validator {
	var found *validator.Validator
	c.iterate(func(v *validator.Validator) (stop bool) {
		if v.Address().EqualsTo(addr) {
			found = v
			return true
		}
		return false
	})
	return found
}

// IsProposer checks if the given address is the proposer for the specified round.
func (c *committee) IsProposer(addr crypto.Address, round int16) bool {
	p := c.proposer(round)
	return p.Address().EqualsTo(addr)
}

// Proposer returns an instance of the proposer validator for the specified round.
// A cloned instance of the proposer is returned to avoid modification of the original object.
func (c *committee) Proposer(round int16) *validator.Validator {
	return cloneValidator(c.proposer(round))
}

func (c *committee) proposer(round int16) *validator.Validator {
	pos := c.proposerPos
	for i := 0; i < int(round); i++ {
		pos = pos.Next()
		if pos == nil {
			pos = c.validatorList.Front()
		}
	}

	return pos.Value.(*validator.Validator)
}

func (c *committee) Committers() []int32 {
	committers := make([]int32, c.validatorList.Len())
	i := 0
	c.iterate(func(v *validator.Validator) (stop bool) {
		committers[i] = v.Number()
		i++
		return false
	})

	return committers
}

func (c *committee) Size() int {
	return c.validatorList.Len()
}

func (c *committee) String() string {
	var builder strings.Builder

	builder.WriteString("[ ")
	for _, v := range c.Validators() {
		builder.WriteString(fmt.Sprintf("%v(%v)", v.Number(), v.LastSortitionHeight()))
		if c.IsProposer(v.Address(), 0) {
			builder.WriteString("*")
		}
		builder.WriteString(" ")
	}
	builder.WriteString("]")

	str := builder.String()

	return str
}

// iterate uses for easy iteration over validators in list.
func (c *committee) iterate(consumer func(*validator.Validator) (stop bool)) {
	for e := c.validatorList.Front(); e != nil; e = e.Next() {
		if consumer(e.Value.(*validator.Validator)) {
			return
		}
	}
}
