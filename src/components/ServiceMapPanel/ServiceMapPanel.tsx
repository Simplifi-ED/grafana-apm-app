import React, { useEffect, useMemo } from 'react';
import { EmbeddedScene, PanelBuilders, SceneDataNode, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';
import { getDefaultTimeRange, LoadingState, type DataFrame, type PanelData, type GrafanaTheme2 } from '@grafana/data';
import {
  LayoutAlgorithm,
  ZoomMode,
} from '@grafana/schema/dist/esm/raw/composable/nodegraph/panelcfg/x/NodeGraphPanelCfg_types.gen';
import { css } from '@emotion/css';
import { useStyles2 } from '@grafana/ui';

function panelDataFromFrames(series: DataFrame[]): PanelData {
  return {
    state: LoadingState.Done,
    series,
    timeRange: getDefaultTimeRange(),
  };
}

interface ServiceMapPanelProps {
  frames: DataFrame[];
}

/**
 * Native Grafana Node Graph (force layout) fed by pre-built node/edge DataFrames.
 * Same visualization primitive Tempo Explore uses for service maps.
 */
export function ServiceMapPanel({ frames }: ServiceMapPanelProps) {
  const styles = useStyles2(getStyles);

  const { scene, dataNode } = useMemo(() => {
    const dataNode = new SceneDataNode({
      data: panelDataFromFrames(frames),
    });
    const scene = new EmbeddedScene({
      $data: dataNode,
      body: new SceneFlexLayout({
        direction: 'column',
        children: [
          new SceneFlexItem({
            minHeight: '100%',
            body: PanelBuilders.nodegraph()
              .setTitle('')
              .setOption('layoutAlgorithm', LayoutAlgorithm.Force)
              .setOption('zoomMode', ZoomMode.Cooperative)
              .setDisplayMode('transparent')
              .build(),
          }),
        ],
      }),
    });
    return { scene, dataNode };
    // Scene is created once; frames are pushed via setState below.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional stable scene
  }, []);

  useEffect(() => {
    dataNode.setState({ data: panelDataFromFrames(frames) });
  }, [dataNode, frames]);

  return (
    <div className={styles.root} data-testid="service-map-canvas">
      <scene.Component model={scene} />
    </div>
  );
}

const getStyles = (_theme: GrafanaTheme2) => ({
  root: css`
    width: 100%;
    height: 100%;
    min-height: 320px;

    /* Scenes/VizPanel chrome should fill the parent canvas. */
    & > div {
      height: 100%;
    }
  `,
});
