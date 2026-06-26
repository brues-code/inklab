import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { BrowseItemsByClassAndSlot } from '../../utils/databaseApi'
import { GetItemClasses, BrowseItemsByClass, GetItemStatTypes } from '../../../wailsjs/go/main/App'
import { GetItemDetail, IsFavorite } from '../../services/api'

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
    enabled: boolean
) => {
    const useSlot = selectedSlot !== null && selectedSlot.inventoryType !== -1
    return useQuery({
        queryKey: queryKeys.items(selectedClass?.class, selectedSubClass?.subClass, useSlot ? selectedSlot!.inventoryType : 'all'),
        queryFn: () =>
            useSlot
                ? BrowseItemsByClassAndSlot(selectedClass!.class, selectedSubClass!.subClass, selectedSlot!.inventoryType, '')
                : BrowseItemsByClass(selectedClass!.class, selectedSubClass!.subClass, ''),
        enabled,
    })
}

export const useItemDetail = (entry: number) =>
    useQuery({ queryKey: queryKeys.itemDetail(entry), queryFn: () => GetItemDetail(entry), enabled: !!entry })

export const useItemFavorite = (entry: number) =>
    useQuery({ queryKey: queryKeys.itemFavorite(entry), queryFn: () => IsFavorite(entry), enabled: !!entry })
