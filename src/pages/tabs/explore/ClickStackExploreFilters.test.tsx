import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { ClickStackExploreFilters, LOG_FIELD_KEYS } from './ClickStackExploreFilters';

describe('ClickStackExploreFilters', () => {
  it('adds and removes field chips', () => {
    const onFieldsChange = jest.fn();
    const { rerender } = render(
      <ClickStackExploreFilters
        fieldKeys={LOG_FIELD_KEYS}
        fields={[]}
        onFieldsChange={onFieldsChange}
        sqlWhere=""
        onSqlWhereChange={jest.fn()}
        sqlHint="hint"
      />
    );

    fireEvent.change(screen.getByPlaceholderText('value'), { target: { value: 'api-pod' } });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    expect(onFieldsChange).toHaveBeenCalledWith([{ key: 'k8s.pod.name', value: 'api-pod' }]);

    rerender(
      <ClickStackExploreFilters
        fieldKeys={LOG_FIELD_KEYS}
        fields={[{ key: 'k8s.pod.name', value: 'api-pod' }]}
        onFieldsChange={onFieldsChange}
        sqlWhere=""
        onSqlWhereChange={jest.fn()}
        sqlHint="hint"
      />
    );
    expect(screen.getByText('k8s.pod.name=api-pod')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Remove filter' }));
    expect(onFieldsChange).toHaveBeenLastCalledWith([]);
  });

  it('applies Advanced SQL via the hatch', () => {
    const onSqlWhereChange = jest.fn();
    render(
      <ClickStackExploreFilters
        fieldKeys={LOG_FIELD_KEYS}
        fields={[]}
        onFieldsChange={jest.fn()}
        sqlWhere=""
        onSqlWhereChange={onSqlWhereChange}
        sqlHint="hint"
      />
    );
    fireEvent.click(screen.getByRole('button', { name: /Advanced SQL/i }));
    fireEvent.change(screen.getByPlaceholderText(/SeverityText/), {
      target: { value: "SeverityText = 'error'" },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Apply SQL' }));
    expect(onSqlWhereChange).toHaveBeenCalledWith("SeverityText = 'error'");
  });
});
