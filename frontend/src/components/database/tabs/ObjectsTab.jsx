import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import {
    SidebarPanel,
    ContentPanel,
    ScrollList,
    SectionHeader,
    ListItem,
    EntityIcon,
} from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useObjectTypes, useObjectsByType } from '../../../hooks/queries/objects'

const OBJECT_COLOR = '#00B4FF'

function ObjectsTab({ onNavigate }) {
    const [selectedObjectType, setSelectedObjectType] = useStickyState(
        'objects.selectedObjectType',
        null,
    )

    const [typeFilter, setTypeFilter] = useStickyState('objects.typeFilter', '')
    const [objectFilter, setObjectFilter] = useStickyState('objects.objectFilter', '')

    const typesQuery = useObjectTypes()
    const objectsQuery = useObjectsByType(selectedObjectType?.id, selectedObjectType != null)

    const objectTypes = typesQuery.data || []
    const objects = objectsQuery.data || []

    const pickType = (type) => {
        setSelectedObjectType(type)
        setObjectFilter('')
    }

    const filteredTypes = useMemo(
        () => filterItems(objectTypes, typeFilter),
        [objectTypes, typeFilter],
    )
    const filteredObjects = useMemo(
        () => filterItems(objects, objectFilter),
        [objects, objectFilter],
    )

    return (
        <>
            {/* Object Types */}
            <SidebarPanel className="col-span-1">
                <SectionHeader
                    title={`Object Types (${filteredTypes.length})`}
                    placeholder="Filter types..."
                    onFilterChange={setTypeFilter}
                />
                <ScrollList>
                    {typesQuery.isLoading && (
                        <div className="animate-pulse p-4 text-center italic text-wow-gold">
                            Loading types...
                        </div>
                    )}
                    {filteredTypes.map((type) => (
                        <ListItem
                            key={type.id}
                            active={selectedObjectType?.id === type.id}
                            onClick={() => pickType(type)}
                        >
                            <span className="flex w-full justify-between">
                                <span>{type.name}</span>
                                <span className="text-xs text-gray-600">({type.count})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Objects List */}
            <ContentPanel className="col-span-3">
                <SectionHeader
                    title={
                        selectedObjectType
                            ? `${selectedObjectType.name} (${filteredObjects.length})`
                            : 'Select a Type'
                    }
                    placeholder="Filter objects..."
                    onFilterChange={setObjectFilter}
                />

                {objectsQuery.isLoading && (
                    <div className="flex flex-1 animate-pulse items-center justify-center italic text-wow-gold">
                        Loading objects...
                    </div>
                )}

                {!objectsQuery.isLoading && objects.length > 0 && (
                    <ScrollList className="space-y-1 p-2">
                        {filteredObjects.map((obj) => (
                            <div
                                key={obj.entry}
                                className="flex cursor-pointer items-center gap-3 rounded-r border-l-[3px] bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                style={{ borderLeftColor: OBJECT_COLOR }}
                                onClick={() => onNavigate?.('object', obj.entry)}
                            >
                                <EntityIcon label="OBJ" color={OBJECT_COLOR} size="md" />

                                <span className="min-w-[50px] font-mono text-[11px] text-gray-600">
                                    [{obj.entry}]
                                </span>

                                <span
                                    className="flex-1 truncate font-bold"
                                    style={{ color: OBJECT_COLOR }}
                                >
                                    {obj.name}
                                </span>

                                <span className="ml-auto text-xs text-gray-500">
                                    Type: {obj.typeName || obj.type} | Size: {obj.size.toFixed(1)}
                                </span>
                            </div>
                        ))}
                    </ScrollList>
                )}

                {!selectedObjectType && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Select an object type to browse
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default ObjectsTab
