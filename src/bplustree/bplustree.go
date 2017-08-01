package bplustree

import (
  "unsafe"
)

const (
  LEAFPAGE = 0
  INTERPAGE = 1
)
/*
** An instance of this object represents a single database file.
**
** A single database file can be in use at the same time by two
** or more database connections.  When two or more connections are
** sharing the same database file, each connection has it own
** private Btree object for the file and each of those Btrees points
** to this one BPlusTree object.
*/
type BPlusTree struct {
  pPager *Pager           /* The page cache */
  page *PgHead          /* First page of the database */
}


/* The basic idea is that each page of the file contains N database
** entries and N+1 pointers to subpages.
**
**   --------------------------------------------------------------
**   |  Ptr(0) | Key(0) | Ptr(1) | Key(1) | ... | Key(N) | Ptr(N) |
**   --------------------------------------------------------------
*/
type Cell struct {
  ptr      uint32      /* page number or Offset of the page start of a payload */
  key      uint32      /* The key for Payload*/
}

/* DocId1 DocId2 ...
**   -----------------------------------------------------------------
**   |  key | DocSiz | DocId1 | DocId3 | ... | DocId(N-1) | DocId(N) |
**   -----------------------------------------------------------------
 */
type Payload struct {
  key     uint32             /* value in the unpacked key */
  size    uint16             /* Number of values.  Might be zero */
  entrys  *[]byte            /* fot data compress */
}

func (bpTree *BPlusTree) Open(dbName string) {
  pPager := &Pager{}
  pPager.Open(dbName)
  bpTree.pPager = pPager
  _, pgHdr := pPager.Fatch(0)
  bpTree.page = pgHdr.GetPageHeader()
}

func (bpTree *BPlusTree) Insert(pl *PlayLoad) {
  offset, pg := Search(pl.key)
  if offset != nil {
    return
  }

  ok, key, newpg := insert(pl, pg)
  if ok != nil {
    return
  }

  ppg := bpTree.hm[pg.parent()]

  for {
    ok, key, newpg = insert(&Cell{key: key,ptr: newpg.phno}, ppg)
    if ok != nil {
      return
    }

    if ppg.pgno == bpTree.page.pgno {
      // alloc new root page for bplustree and update bplustree page
      rootpage := &MemPage{}
      bpTree.page = rootpage
      // insert new page cell
      insert(&Cell{key: key,ptr: newpg.phno}, rootpage)

      // insert origin page cell
      insert(&Cell{key: key,ptr: ppg.phno}, rootpage)
      return
    }
    ppg = bpTree.hm[ppg.parent()]
  }
}

func (bpTree *BPlusTree) Search(key int) (uint16, *MemPage) {
  curr := bpTree.page
  for {
    switch t := curr.flag {
    case LEAFPAGE:
      offset, ok := find(curr, key)
      if !ok {
        return nil, curr
      }
      return offset, curr
    case INTERPAGE:
      pgno, _ := find(key)
      curr = bpTree.hm[pgno]
      // pager should load page and cached
    default:
      panic("no such flag!")
    }
  }
}

func (p *MemPage) insert(data interface{}) (bool, uint32, *MemPage){
  ok := p.full(data)
  if !ok {
    return true, nil, nil
  }

  //key, newpg :=split(pg)
  newpg := newpage()
  //update page info
  newpg.maxkey = p.maxkey
  p.maxkey = ((*Cell)p.cell)[ncell/2].key
  p.ncell = ncell/2

  return false, key, newpg
}

func (p *MemPage) find(key int) (int, bool) {
  cmp := func (i int) bool {
    return p.cell[i].key >= key
  }

  i := sort.Search(p.ncell, cmp)

  if p.flag == INTERPAGE {
    return p.cell[i].ptr, true
  }

  if i <= p.ncell && p.cell[i].key == key {
    return p.cell[i].ptr, true
  }

  return nil, false
}

func newpage() *MemPage{
  page := &MemPage{}
  // alloc page in cache
  // page.cell = cache.allocpage()[100]
}

func (p *MemPage) parent() uint32 {
  return p.ppgno
}

func (p *MemPage) setparent(uint32 pgno) {
  p.ppgno = pgno
}

func (p *MemPage) full(data interface{}) bool {
  switch data.(type){
  case *Cell:
    if p.flag == INTERPAGE {
      return p.free >= size(Cell)
    }
    panic("full error")
  case *PlayLoad:
    if p.flag == LEAFPAGE {
      return p.free >= (pl.size + size(Cell))
    }
    panic("full error")
  }
}
