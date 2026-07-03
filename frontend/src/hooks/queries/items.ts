import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { BrowseItemsByClassAndSlot } from '../../utils/databaseApi'
import {
    GetItemClasses,
    BrowseItemsByClass,
    GetItemStatTypes,
    BrowseItems,
    GetItemRandomSuffixes,
} from '../../../wailsjs/go/main/App'
import { GetItemDetail, IsFavorite } from '../../services/api'
import { models } from '../../../wailsjs/go/models'

export const useItemClasses = () =>
    useQuery({ queryKey: queryKeys.itemClasses, queryFn: GetItemClasses, staleTime: Infinity })

// Stat types present in the item data (id + display name), for the filter's stat
// dropdown. Static for a session.
export const useItemStatTypes = () =>
    useQuery({ queryKey: queryKeys.itemStatTypes, queryFn: GetItemStatTypes, staleTime: Infinity })

type ItemClass = { class: number }
type ItemSubClass = { subClass: number }
type ItemSlot = { inventoryType: number }

// Browse items for the current class/subclass/(slot). A specific slot uses the
// slot-aware query; "All Slots" (inventoryType -1) or non-slot classes use the
// class+subclass query. Non-null assertions are safe: `enabled` gates the fetch
// until the needed selections exist.
export const useItems = (
    selectedClass: ItemClass | null,
    selectedSubClass: ItemSubClass | null,
    selectedSlot: ItemSlot | null,
    enabled: boolean,
) => {
    const useSlot = selectedSlot !== null && selectedSlot.inventoryType !== -1
    return useQuery({
        queryKey: queryKeys.items(
            selectedClass?.class,
            selectedSubClass?.subClass,
            useSlot ? selectedSlot!.inventoryType : 'all',
        ),
        queryFn: () =>
            useSlot
                ? BrowseItemsByClassAndSlot(
                      selectedClass!.class,
                      selectedSubClass!.subClass,
                      selectedSlot!.inventoryType,
                      '',
                  )
                : BrowseItemsByClass(selectedClass!.class, selectedSubClass!.subClass, ''),
        enabled,
    })
}

// Paginated, filtered item browse powering the Items page. The full filter
// (including paging + sort) is the query key, so each distinct view is cached;
// placeholderData keeps the previous page visible while the next loads (no
// flicker on sort/page changes).
export const useBrowseItems = (filter: models.SearchFilter) =>
    useQuery({
        queryKey: queryKeys.itemBrowse(filter),
        queryFn: () => BrowseItems(filter),
        placeholderData: keepPreviousData,
    })

export const useItemDetail = (entry: number) =>
    useQuery({
        queryKey: queryKeys.itemDetail(entry),
        queryFn: () => GetItemDetail(entry),
        enabled: !!entry,
    })

export const useItemFavorite = (entry: number) =>
    useQuery({
        queryKey: queryKeys.itemFavorite(entry),
        queryFn: () => IsFavorite(entry),
        enabled: !!entry,
    })

// Possible random suffixes ("of the Monkey") for an item with a random
// property, with stat ranges and roll chances. Empty for normal items.
export const useItemRandomSuffixes = (entry: number) =>
    useQuery({
        queryKey: queryKeys.itemRandomSuffixes(entry),
        queryFn: () => GetItemRandomSuffixes(entry),
        enabled: !!entry,
        staleTime: Infinity,
    })
